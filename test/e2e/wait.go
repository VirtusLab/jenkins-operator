package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/base/resources"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	_, err := WaitUntilJenkinsConditionTrue(retryInterval, 150, jenkins, func(jenkins *virtuslabv1alpha1.Jenkins) bool {
		t.Logf("Current Jenkins status '%+v'", jenkins.Status)
		return jenkins.Status.BaseConfigurationCompletedTime != nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Jenkins pod is running")
}

func waitForRecreateJenkinsMasterPod(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) {
	err := wait.Poll(retryInterval, 30*retryInterval, func() (bool, error) {
		lo := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(resources.BuildResourceLabels(jenkins)).String(),
		}
		podList, err := framework.Global.KubeClient.CoreV1().Pods(jenkins.ObjectMeta.Namespace).List(lo)
		if err != nil {
			return false, err
		}
		if len(podList.Items) != 1 {
			return false, nil
		}

		return podList.Items[0].DeletionTimestamp == nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Jenkins pod has been recreated")
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
