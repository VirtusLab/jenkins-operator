package e2e

import (
	"context"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"k8s.io/apimachinery/pkg/types"
)

func TestJenkinsMasterPodRestart(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	jenkins := createJenkinsCR(t, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	restartJenkinsMasterPod(t, jenkins)
	waitForRecreateJenkinsMasterPod(t, jenkins)
	checkBaseConfigurationCompleteTimeIsNotSet(t, jenkins)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
}

func checkBaseConfigurationCompleteTimeIsNotSet(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) {
	jenkinsStatus := &virtuslabv1alpha1.Jenkins{}
	namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}
	err := framework.Global.Client.Get(context.TODO(), namespaceName, jenkinsStatus)
	if err != nil {
		t.Fatal(err)
	}
	if jenkinsStatus.Status.BaseConfigurationCompletedTime != nil {
		t.Fatalf("Status.BaseConfigurationCompletedTime is set after pod restart, status %+v", jenkinsStatus.Status)
	}
}
