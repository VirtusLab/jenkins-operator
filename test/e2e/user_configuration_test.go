package e2e

import (
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/user/seedjobs"
	"github.com/bndr/gojenkins"
	"k8s.io/apimachinery/pkg/util/wait"
	"testing"
	"time"
)

func TestUserConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	jenkins := createJenkinsCRWithSeedJob(t, namespace)
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	client := verifyJenkinsAPIConnection(t, jenkins)
	verifyJenkinsSeedJobs(t, client)
}

func verifyJenkinsSeedJobs(t *testing.T, client *gojenkins.Jenkins) {
	// check if job has been configured and executed successfully
	err := wait.Poll(time.Second*10, time.Minute*2, func() (bool, error) {
		t.Logf("Attempting to get seed job status '%v'", seedjobs.ConfigureSeedJobsName)
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
		t.Fatalf("couldn't get seed job '%v'", err)
	}
}
