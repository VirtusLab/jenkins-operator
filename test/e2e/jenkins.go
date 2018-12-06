package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/base/resources"

	"github.com/bndr/gojenkins"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

func getJenkinsMasterPod(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) *v1.Pod {
	lo := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(resources.BuildResourceLabels(jenkins)).String(),
	}
	podList, err := framework.Global.KubeClient.CoreV1().Pods(jenkins.ObjectMeta.Namespace).List(lo)
	if err != nil {
		t.Fatal(err)
	}
	if len(podList.Items) != 1 {
		t.Fatalf("Jenkins pod not found, pod list: %+v", podList)
	}
	return &podList.Items[0]
}

func createJenkinsAPIClient(jenkins *virtuslabv1alpha1.Jenkins) (*gojenkins.Jenkins, error) {
	adminSecret := &v1.Secret{}
	namespacedName := types.NamespacedName{Namespace: jenkins.Namespace, Name: resources.GetOperatorCredentialsSecretName(jenkins)}
	if err := framework.Global.Client.Get(context.TODO(), namespacedName, adminSecret); err != nil {
		return nil, err
	}

	jenkinsAPIURL, err := jenkinsclient.BuildJenkinsAPIUrl(jenkins.ObjectMeta.Namespace, resources.GetResourceName(jenkins), resources.HTTPPortInt, true, true)
	if err != nil {
		return nil, err
	}

	jenkinsClient := gojenkins.CreateJenkins(
		jenkinsAPIURL,
		string(adminSecret.Data[resources.OperatorCredentialsSecretUserNameKey]),
		string(adminSecret.Data[resources.OperatorCredentialsSecretTokenKey]),
	)
	if _, err := jenkinsClient.Init(); err != nil {
		return nil, err
	}

	status, err := jenkinsClient.Poll()
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("invalid status code returned: %d", status)
	}

	return jenkinsClient, nil
}
