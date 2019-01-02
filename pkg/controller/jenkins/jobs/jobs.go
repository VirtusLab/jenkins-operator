package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/log"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// SuccessStatus - the build had no errors
	SuccessStatus = "success"
	// UnstableStatus - the build had some errors but they were not fatal. For example, some tests failed
	UnstableStatus = "unstable"
	// NotBuildStatus - this status code is used in a multi-stage build (like maven2) where a problem in earlier stage prevented later stages from building
	NotBuildStatus = "not_build"
	// FailureStatus - the build had a fatal error
	FailureStatus = "failure"
	// AbortedStatus - the build was manually aborted
	AbortedStatus = "aborted"
	// RunningStatus - this is custom build status for running build, not present in jenkins build result
	RunningStatus = "running"
	// ExpiredStatus - this is custom build status for expired build, not present in jenkins build result
	ExpiredStatus = "expired"
	// MaxBuildRetires - determines max amount of retires for failed build
	MaxBuildRetires = 3
)

var (
	// ErrorEmptyJenkinsCR - this is custom error returned when jenkins custom resource is empty
	ErrorEmptyJenkinsCR = errors.New("empty jenkins cr")
	// ErrorUnexpectedBuildStatus - this is custom error returned when jenkins build has unexpected status
	ErrorUnexpectedBuildStatus = errors.New("unexpected build status")
	// ErrorBuildFailed - this is custom error returned when jenkins build has failed
	ErrorBuildFailed = errors.New("build failed")
	// ErrorAbortBuildFailed - this is custom error returned when jenkins build couldn't be aborted
	ErrorAbortBuildFailed = errors.New("build abort failed")
	// ErrorUncoverBuildFailed - this is custom error returned when jenkins build has failed and cannot be recovered
	ErrorUncoverBuildFailed = errors.New("build failed and cannot be recovered")
	// ErrorNotFound - this is error returned when jenkins build couldn't be found
	ErrorNotFound = errors.New("404")
)

// Jobs defines Jobs API tailored for operator sdk
type Jobs struct {
	jenkinsClient client.Jenkins
	logger        logr.Logger
	k8sClient     k8s.Client
}

// New creates jobs client
func New(jenkinsClient client.Jenkins, k8sClient k8s.Client, logger logr.Logger) *Jobs {
	return &Jobs{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
	}
}

// EnsureBuildJob function takes care of jenkins build lifecycle according to the lifecycle of reconciliation loop
// implementation guarantees that jenkins build can be properly handled even after operator pod restart
// entire state is saved in Jenkins.Status.Builds section
// function return 'true' when build finished successfully or false when reconciliation loop should requeue this function
// preserveStatus determines that build won't be removed from Jenkins.Status.Builds section
func (jobs *Jobs) EnsureBuildJob(name, hash string, parameters map[string]string, jenkins *virtuslabv1alpha1.Jenkins, preserveStatus bool) (done bool, err error) {
	jobs.logger.Info(fmt.Sprintf("Ensuring build, name:'%s' hash:'%s'", name, hash))

	build, err := jobs.getBuildFromStatus(name, hash, jenkins)
	if err != nil {
		return false, err
	}

	if build != nil {
		jobs.logger.Info(fmt.Sprintf("Build exists in status, name:'%s' hash:'%s' status: '%s'", name, hash, build.Status))
		switch strings.ToLower(build.Status) {
		case SuccessStatus:
			return jobs.ensureSuccessBuild(*build, jenkins, preserveStatus)
		case RunningStatus:
			return jobs.ensureRunningBuild(*build, jenkins, preserveStatus)
		case UnstableStatus, NotBuildStatus, FailureStatus, AbortedStatus:
			return jobs.ensureFailedBuild(*build, jenkins, parameters, preserveStatus)
		case ExpiredStatus:
			return jobs.ensureExpiredBuild(*build, jenkins, preserveStatus)
		default:
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Unexpected build status, name:'%s' hash:'%s' status:'%s'", name, hash, build.Status))
			return false, ErrorUnexpectedBuildStatus
		}
	}

	// build is run first time - build job and update status
	jobs.logger.Info(fmt.Sprintf("Build doesn't exist, running and updating status, name:'%s' hash:'%s'", name, hash))
	created := metav1.Now()
	newBuild := virtuslabv1alpha1.Build{
		Name:       name,
		Hash:       hash,
		CreateTime: &created,
	}
	return jobs.buildJob(newBuild, parameters, jenkins)
}

func (jobs *Jobs) getBuildFromStatus(name string, hash string, jenkins *virtuslabv1alpha1.Jenkins) (*virtuslabv1alpha1.Build, error) {
	if jenkins != nil {
		builds := jenkins.Status.Builds
		for _, build := range builds {
			if build.Name == name && build.Hash == hash {
				return &build, nil
			}
		}
	}
	return nil, nil
}

func (jobs *Jobs) ensureSuccessBuild(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins, preserveStatus bool) (bool, error) {
	if jenkins == nil {
		jobs.logger.V(log.VWarn).Info("Jenkins CR is empty")
		return false, ErrorEmptyJenkinsCR
	}

	jobs.logger.Info(fmt.Sprintf("Ensuring success build, name:'%s' hash:'%s'", build.Name, build.Hash))

	if !preserveStatus {
		err := jobs.removeBuildFromStatus(build, jenkins)
		jobs.logger.Info(fmt.Sprintf("Removing build from status, name:'%s' hash:'%s'", build.Name, build.Hash))
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, name:'%s' hash:'%s'", build.Name, build.Hash))
			return false, err
		}
	}
	return true, nil
}

func (jobs *Jobs) ensureRunningBuild(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins, preserveStatus bool) (bool, error) {
	if jenkins == nil {
		jobs.logger.V(log.VWarn).Info("Jenkins CR is empty")
		return false, ErrorEmptyJenkinsCR
	}

	jobs.logger.Info(fmt.Sprintf("Ensuring running build, name:'%s' hash:'%s'", build.Name, build.Hash))
	// FIXME (antoniaklja) implement build expiration

	jenkinsBuild, err := jobs.jenkinsClient.GetBuild(build.Name, build.Number)
	if isNotFoundError(err) {
		jobs.logger.Info(fmt.Sprintf("Build still running , name:'%s' hash:'%s'", build.Name, build.Hash))
		return false, nil
	} else if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't get jenkins build, name:'%s' number:'%d'", build.Name, build.Number))
		return false, err
	}

	if jenkinsBuild.GetResult() != "" {
		build.Status = strings.ToLower(jenkinsBuild.GetResult())
	}

	jobs.logger.Info(fmt.Sprintf("Updating build status, name:'%s' hash:'%s' status:'%s'", build.Name, build.Hash, build.Status))
	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't update build status, name:'%s' hash:'%s'", build.Name, build.Hash))
		return false, err
	}

	if build.Status == SuccessStatus {
		jobs.logger.Info(fmt.Sprintf("Build finished successfully, name:'%s' hash:'%s' status:'%s'", build.Name, build.Hash, build.Status))
		if !preserveStatus {
			jobs.logger.Info(fmt.Sprintf("Removing build from status, name:'%s' hash:'%s'", build.Name, build.Hash))
			err := jobs.removeBuildFromStatus(build, jenkins)
			if err != nil {
				jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, name:'%s' hash:'%s'", build.Name, build.Hash))
				return false, err
			}
		}
		return true, nil
	}

	return false, nil
}

func (jobs *Jobs) ensureFailedBuild(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins, parameters map[string]string, preserveStatus bool) (bool, error) {
	if jenkins == nil {
		jobs.logger.V(log.VWarn).Info("Jenkins CR is empty")
		return false, ErrorEmptyJenkinsCR
	}

	jobs.logger.Info(fmt.Sprintf("Ensuring failed build, name:'%s' hash:'%s' status: '%s'", build.Name, build.Hash, build.Status))

	if build.Retires < MaxBuildRetires {
		jobs.logger.Info(fmt.Sprintf("Retrying build, name:'%s' hash:'%s' retries: '%d'", build.Name, build.Hash, build.Retires))
		build.Retires = build.Retires + 1
		_, err := jobs.buildJob(build, parameters, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't retry build, name:'%s' hash:'%s'", build.Name, build.Hash))
			return false, err
		}
		return false, ErrorBuildFailed
	}

	jobs.logger.Info(fmt.Sprintf("The retries limit was reached , name:'%s' hash:'%s' retries: '%d'", build.Name, build.Hash, build.Retires))

	if !preserveStatus {
		jobs.logger.Info(fmt.Sprintf("Removing build from status, name:'%s' hash:'%s'", build.Name, build.Hash))
		err := jobs.removeBuildFromStatus(build, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, name:'%s' hash:'%s'", build.Name, build.Hash))
			return false, err
		}
	}
	return false, ErrorUncoverBuildFailed
}

func (jobs *Jobs) ensureExpiredBuild(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins, preserveStatus bool) (bool, error) {
	if jenkins == nil {
		jobs.logger.V(log.VWarn).Info("Jenkins CR is empty")
		return false, ErrorEmptyJenkinsCR
	}

	jobs.logger.Info(fmt.Sprintf("Ensuring expired build, name:'%s' hash:'%s' status: '%s'", build.Name, build.Hash, build.Status))

	jenkinsBuild, err := jobs.jenkinsClient.GetBuild(build.Name, build.Number)
	if err != nil {
		return false, err
	}

	_, err = jenkinsBuild.Stop()
	if err != nil {
		return false, err
	}

	jenkinsBuild, err = jobs.jenkinsClient.GetBuild(build.Name, build.Number)
	if err != nil {
		return false, err
	}

	if jenkinsBuild.GetResult() != AbortedStatus {
		return false, ErrorAbortBuildFailed
	}

	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		return false, err
	}

	// TODO(antoniaklja) clean up k8s resources

	if !preserveStatus {
		jobs.logger.Info(fmt.Sprintf("Removing build from status, name:'%s' hash:'%s'", build.Name, build.Hash))
		err = jobs.removeBuildFromStatus(build, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, name:'%s' hash:'%s'", build.Name, build.Hash))
			return false, err
		}
	}

	return true, nil
}

func (jobs *Jobs) removeBuildFromStatus(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins) error {
	if jenkins == nil {
		return ErrorEmptyJenkinsCR
	}

	builds := make([]virtuslabv1alpha1.Build, len(jenkins.Status.Builds), len(jenkins.Status.Builds))
	for _, existingBuild := range jenkins.Status.Builds {
		if existingBuild.Name != build.Name && existingBuild.Hash != build.Hash {
			builds = append(builds, existingBuild)
		}
	}
	jenkins.Status.Builds = builds
	err := jobs.k8sClient.Update(context.TODO(), jenkins)
	if err != nil {
		return err
	}

	return nil
}

func (jobs *Jobs) buildJob(build virtuslabv1alpha1.Build, parameters map[string]string, jenkins *virtuslabv1alpha1.Jenkins) (bool, error) {
	if jenkins == nil {
		return false, ErrorEmptyJenkinsCR
	}

	jobs.logger.Info(fmt.Sprintf("Running build, name:'%s' hash:'%s'", build.Name, build.Hash))

	number, err := jobs.jenkinsClient.BuildJob(build.Name, parameters)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't run build, name:'%s' hash:'%s' number:'%d'", build.Name, build.Hash, number))
		return false, err
	}

	build.Status = RunningStatus
	build.Number = number

	jobs.logger.Info(fmt.Sprintf("Updating build status, name:'%s' hash:'%s' status:'%s'", build.Name, build.Hash, build.Status))
	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't update build status, name:'%s' hash:'%s'", build.Name, build.Hash))
		return false, err
	}
	return false, nil
}

func (jobs *Jobs) updateBuildStatus(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins) error {
	if jenkins == nil {
		return ErrorEmptyJenkinsCR
	}

	// get index of existing build from status if exists
	buildIndex := -1
	for index, existingBuild := range jenkins.Status.Builds {
		if build.Name == existingBuild.Name && build.Hash == existingBuild.Hash {
			buildIndex = index
		}
	}

	// update build status
	now := metav1.Now()
	build.LastUpdateTime = &now
	if buildIndex >= 0 {
		jenkins.Status.Builds[buildIndex] = build
	} else {
		build.CreateTime = &now
		jenkins.Status.Builds = append(jenkins.Status.Builds, build)
	}
	err := jobs.k8sClient.Update(context.TODO(), jenkins)
	if err != nil {
		return err
	}

	return nil
}

func isNotFoundError(err error) bool {
	if err != nil {
		return err.Error() == ErrorNotFound.Error()
	}
	return false
}
