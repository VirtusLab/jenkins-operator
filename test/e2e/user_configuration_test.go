package e2e

import (
	"testing"
	"time"

	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/user/seedjobs"

	"github.com/bndr/gojenkins"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestUserConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	jenkins := createJenkinsCRWithSeedJob(t, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	client := verifyJenkinsAPIConnection(t, jenkins)
	verifyJenkinsSeedJobs(t, client)
}

func verifyJenkinsSeedJobs(t *testing.T, client *gojenkins.Jenkins) {
	// check if job has been configured and executed successfully
	err := wait.Poll(time.Second*10, time.Minute*2, func() (bool, error) {
		t.Logf("Attempting to get configure seed job status '%v'", seedjobs.ConfigureSeedJobsName)
		seedJob, err := client.GetJob(seedjobs.ConfigureSeedJobsName)
		if err != nil || seedJob == nil {
			return false, nil
		}
		build, err := seedJob.GetLastSuccessfulBuild()
		if err != nil || build == nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("couldn't get configure seed job '%v'", err)
	}

	// WARNING this use case depends on changes in https://github.com/VirtusLab/jenkins-operator-e2e/tree/master/cicd
	seedJobName := "jenkins-operator-e2e-job-dsl-seed" // https://github.com/VirtusLab/jenkins-operator-e2e/blob/master/cicd/jobs/e2e_test_job.jenkins
	err = wait.Poll(time.Second*10, time.Minute*2, func() (bool, error) {
		t.Logf("Attempting to verify if seed job has been created '%v'", seedJobName)
		seedJob, err := client.GetJob(seedJobName)
		if err != nil || seedJob == nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("couldn't verify if seed job has been created '%v'", err)
	}
}
