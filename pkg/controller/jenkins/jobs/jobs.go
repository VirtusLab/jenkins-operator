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

var (
	// ErrorUnexpectedBuildStatus - this is custom error returned when jenkins build has unexpected status
	ErrorUnexpectedBuildStatus = errors.New("unexpected build status")
	// ErrorBuildFailed - this is custom error returned when jenkins build has failed
	ErrorBuildFailed = errors.New("build failed")
	// ErrorAbortBuildFailed - this is custom error returned when jenkins build couldn't be aborted
	ErrorAbortBuildFailed = errors.New("build abort failed")
	// ErrorUnrecoverableBuildFailed - this is custom error returned when jenkins build has failed and cannot be recovered
	ErrorUnrecoverableBuildFailed = errors.New("build failed and cannot be recovered")
	// ErrorNotFound - this is error returned when jenkins build couldn't be found
	ErrorNotFound = errors.New("404")
	// BuildRetires - determines max amount of retires for failed build
	BuildRetires = 3
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
func (jobs *Jobs) EnsureBuildJob(jobName, hash string, parameters map[string]string, jenkins *virtuslabv1alpha1.Jenkins, preserveStatus bool) (done bool, err error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring build, name:'%s' hash:'%s'", jobName, hash))

	build, err := jobs.getBuildFromStatus(jobName, hash, jenkins)
	if err != nil {
		return false, err
	}

	if build != nil {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Build exists in status, %+v", build))
		switch build.Status {
		case virtuslabv1alpha1.BuildSuccessStatus:
			return jobs.ensureSuccessBuild(*build, jenkins, preserveStatus)
		case virtuslabv1alpha1.BuildRunningStatus:
			return jobs.ensureRunningBuild(*build, jenkins, preserveStatus)
		case virtuslabv1alpha1.BuildUnstableStatus, virtuslabv1alpha1.BuildNotBuildStatus, virtuslabv1alpha1.BuildFailureStatus, virtuslabv1alpha1.BuildAbortedStatus:
			return jobs.ensureFailedBuild(*build, jenkins, parameters, preserveStatus)
		case virtuslabv1alpha1.BuildExpiredStatus:
			return jobs.ensureExpiredBuild(*build, jenkins, preserveStatus)
		default:
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Unexpected build status, %+v", build))
			return false, ErrorUnexpectedBuildStatus
		}
	}

	// build is run first time - build job and update status
	created := metav1.Now()
	newBuild := virtuslabv1alpha1.Build{
		JobName:    jobName,
		Hash:       hash,
		CreateTime: &created,
	}
	return jobs.buildJob(newBuild, parameters, jenkins)
}

func (jobs *Jobs) getBuildFromStatus(jobName string, hash string, jenkins *virtuslabv1alpha1.Jenkins) (*virtuslabv1alpha1.Build, error) {
	if jenkins != nil {
		builds := jenkins.Status.Builds
		for _, build := range builds {
			if build.JobName == jobName && build.Hash == hash {
				return &build, nil
			}
		}
	}
	return nil, nil
}

func (jobs *Jobs) ensureSuccessBuild(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins, preserveStatus bool) (bool, error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring success build, %+v", build))

	if !preserveStatus {
		err := jobs.removeBuildFromStatus(build, jenkins)
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Removing build from status, %+v", build))
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, %+v", build))
			return false, err
		}
	}
	return true, nil
}

func (jobs *Jobs) ensureRunningBuild(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins, preserveStatus bool) (bool, error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring running build, %+v", build))
	// FIXME (antoniaklja) implement build expiration

	jenkinsBuild, err := jobs.jenkinsClient.GetBuild(build.JobName, build.Number)
	if isNotFoundError(err) {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Build still running , %+v", build))
		return false, nil
	} else if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't get jenkins build, %+v", build))
		return false, err
	}

	if jenkinsBuild.GetResult() != "" {
		build.Status = virtuslabv1alpha1.BuildStatus(strings.ToLower(jenkinsBuild.GetResult()))
	}

	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't update build status, %+v", build))
		return false, err
	}

	if build.Status == virtuslabv1alpha1.BuildSuccessStatus {
		jobs.logger.Info(fmt.Sprintf("Build finished successfully, %+v", build))
		return true, nil
	}

	if build.Status == virtuslabv1alpha1.BuildFailureStatus || build.Status == virtuslabv1alpha1.BuildUnstableStatus ||
		build.Status == virtuslabv1alpha1.BuildNotBuildStatus || build.Status == virtuslabv1alpha1.BuildAbortedStatus {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Build failed, %+v", build))
		return false, ErrorBuildFailed
	}

	return false, nil
}

func (jobs *Jobs) ensureFailedBuild(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins, parameters map[string]string, preserveStatus bool) (bool, error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring failed build, %+v", build))

	if build.Retires < BuildRetires {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Retrying build, %+v", build))
		build.Retires = build.Retires + 1
		_, err := jobs.buildJob(build, parameters, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't retry build, %+v", build))
			return false, err
		}
		return false, nil
	}

	jobs.logger.V(log.VWarn).Info(fmt.Sprintf("The retries limit was reached , %+v", build))

	if !preserveStatus {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Removing build from status, %+v", build))
		err := jobs.removeBuildFromStatus(build, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, %+v", build))
			return false, err
		}
	}
	return false, ErrorUnrecoverableBuildFailed
}

func (jobs *Jobs) ensureExpiredBuild(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins, preserveStatus bool) (bool, error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring expired build, %+v", build))

	jenkinsBuild, err := jobs.jenkinsClient.GetBuild(build.JobName, build.Number)
	if err != nil {
		return false, err
	}

	_, err = jenkinsBuild.Stop()
	if err != nil {
		return false, err
	}

	jenkinsBuild, err = jobs.jenkinsClient.GetBuild(build.JobName, build.Number)
	if err != nil {
		return false, err
	}

	if virtuslabv1alpha1.BuildStatus(jenkinsBuild.GetResult()) != virtuslabv1alpha1.BuildAbortedStatus {
		return false, ErrorAbortBuildFailed
	}

	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		return false, err
	}

	// TODO(antoniaklja) clean up k8s resources

	if !preserveStatus {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Removing build from status, %+v", build))
		err = jobs.removeBuildFromStatus(build, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, %+v", build))
			return false, err
		}
	}

	return true, nil
}

func (jobs *Jobs) removeBuildFromStatus(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins) error {
	builds := make([]virtuslabv1alpha1.Build, len(jenkins.Status.Builds))
	for _, existingBuild := range jenkins.Status.Builds {
		if existingBuild.JobName != build.JobName && existingBuild.Hash != build.Hash {
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
	jobs.logger.Info(fmt.Sprintf("Running job, %+v", build))
	job, err := jobs.jenkinsClient.GetJob(build.JobName)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't find jenkins job, %+v", build))
		return false, err
	}
	nextBuildNumber := job.GetDetails().NextBuildNumber

	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Running build, %+v", build))
	_, err = jobs.jenkinsClient.BuildJob(build.JobName, parameters)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't run build, %+v", build))
		return false, err
	}

	build.Status = virtuslabv1alpha1.BuildRunningStatus
	build.Number = nextBuildNumber

	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't update build status, %+v", build))
		return false, err
	}
	return false, nil
}

func (jobs *Jobs) updateBuildStatus(build virtuslabv1alpha1.Build, jenkins *virtuslabv1alpha1.Jenkins) error {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Updating build status, %+v", build))
	// get index of existing build from status if exists
	buildIndex := -1
	for index, existingBuild := range jenkins.Status.Builds {
		if build.JobName == existingBuild.JobName && build.Hash == existingBuild.Hash {
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
