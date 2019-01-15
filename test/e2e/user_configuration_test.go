package e2e

import (
	"context"
	"testing"
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/user/seedjobs"

	"github.com/bndr/gojenkins"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestUserConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	// base
	jenkins := createJenkinsCRWithSeedJob(t, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	client := verifyJenkinsAPIConnection(t, jenkins)

	// user
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	verifyJenkinsSeedJobs(t, client, jenkins)
}

func verifyJenkinsSeedJobs(t *testing.T, client *gojenkins.Jenkins, jenkins *virtuslabv1alpha1.Jenkins) {
	t.Logf("Attempting to get configure seed job status '%v'", seedjobs.ConfigureSeedJobsName)

	configureSeedJobs, err := client.GetJob(seedjobs.ConfigureSeedJobsName)
	assert.NoError(t, err)
	assert.NotNil(t, configureSeedJobs)
	build, err := configureSeedJobs.GetLastSuccessfulBuild()
	assert.NoError(t, err)
	assert.NotNil(t, build)

	seedJobName := "jenkins-operator-configure-seed-job"
	t.Logf("Attempting to verify if seed job has been created '%v'", seedJobName)
	seedJob, err := client.GetJob(seedJobName)
	assert.NoError(t, err)
	assert.NotNil(t, seedJob)

	build, err = seedJob.GetLastSuccessfulBuild()
	assert.NoError(t, err)
	assert.NotNil(t, build)

	err = framework.Global.Client.Get(context.TODO(), types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}, jenkins)
	assert.NoError(t, err, "couldn't get jenkins custom resource")
	assert.NotNil(t, jenkins.Status.Builds)
	assert.NotEmpty(t, jenkins.Status.Builds)

	jobCreatedByDSLPluginName := "build-jenkins-operator"
	err = wait.Poll(time.Second*10, time.Minute*2, func() (bool, error) {
		t.Logf("Attempting to verify if job '%s' has been created ", jobCreatedByDSLPluginName)
		seedJob, err := client.GetJob(jobCreatedByDSLPluginName)
		if err != nil || seedJob == nil {
			return false, nil
		}
		return true, nil
	})
	assert.NoError(t, err)
}
