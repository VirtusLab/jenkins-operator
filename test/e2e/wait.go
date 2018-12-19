package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	retryInterval = time.Second * 5
	timeout       = time.Second * 60
)

// checkConditionFunc is used to check if a condition for the jenkins CR is true
type checkConditionFunc func(*virtuslabv1alpha1.Jenkins) bool

func waitForJenkinsBaseConfigurationToComplete(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) {
	t.Log("Waiting for Jenkins base configuration to complete")
	_, err := WaitUntilJenkinsConditionTrue(retryInterval, 30, jenkins, func(jenkins *virtuslabv1alpha1.Jenkins) bool {
		t.Logf("Current Jenkins status '%+v'", jenkins.Status)
		return jenkins.Status.BaseConfigurationCompletedTime != nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Jenkins pod is running")
}

func waitForJenkinsUserConfigurationToComplete(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) {
	t.Log("Waiting for Jenkins user configuration to complete")
	_, err := WaitUntilJenkinsConditionTrue(retryInterval, 30, jenkins, func(jenkins *virtuslabv1alpha1.Jenkins) bool {
		t.Logf("Current Jenkins status '%+v'", jenkins.Status)
		return jenkins.Status.UserConfigurationCompletedTime != nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Jenkins pod is running")
}

// WaitUntilJenkinsConditionTrue retries until the specified condition check becomes true for the jenkins CR
func WaitUntilJenkinsConditionTrue(retryInterval time.Duration, retries int, jenkins *virtuslabv1alpha1.Jenkins, checkCondition checkConditionFunc) (*virtuslabv1alpha1.Jenkins, error) {
	jenkinsStatus := &virtuslabv1alpha1.Jenkins{}
	err := wait.Poll(retryInterval, time.Duration(retries)*retryInterval, func() (bool, error) {
		namespacedName := types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}
		err := framework.Global.Client.Get(goctx.TODO(), namespacedName, jenkinsStatus)
		if err != nil {
			return false, fmt.Errorf("failed to get CR: %v", err)
		}
		return checkCondition(jenkinsStatus), nil
	})
	if err != nil {
		return nil, err
	}
	return jenkinsStatus, nil
}
