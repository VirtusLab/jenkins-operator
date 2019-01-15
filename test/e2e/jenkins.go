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

func getJenkins(t *testing.T, namespace, name string) *virtuslabv1alpha1.Jenkins {
	jenkins := &virtuslabv1alpha1.Jenkins{}
	namespaceName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := framework.Global.Client.Get(context.TODO(), namespaceName, jenkins); err != nil {
		t.Fatal(err)
	}

	return jenkins
}

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
	namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: resources.GetOperatorCredentialsSecretName(jenkins)}
	if err := framework.Global.Client.Get(context.TODO(), namespaceName, adminSecret); err != nil {
		return nil, err
	}

	jenkinsAPIURL, err := jenkinsclient.BuildJenkinsAPIUrl(jenkins.ObjectMeta.Namespace, resources.GetResourceName(jenkins), resources.HTTPPortInt, true, true)
	if err != nil {
		return nil, err
	}

	jenkinsClient := gojenkins.CreateJenkins(
		nil,
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

func createJenkinsCR(t *testing.T, namespace string) *virtuslabv1alpha1.Jenkins {
	jenkins := &virtuslabv1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e",
			Namespace: namespace,
		},
		Spec: virtuslabv1alpha1.JenkinsSpec{
			Master: virtuslabv1alpha1.JenkinsMaster{
				Image:       "jenkins/jenkins",
				Annotations: map[string]string{"test": "label"},
			},
		},
	}

	t.Logf("Jenkins CR %+v", *jenkins)
	if err := framework.Global.Client.Create(context.TODO(), jenkins, nil); err != nil {
		t.Fatal(err)
	}

	return jenkins
}

func createJenkinsCRWithSeedJob(t *testing.T, namespace string) *virtuslabv1alpha1.Jenkins {
	jenkins := &virtuslabv1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e",
			Namespace: namespace,
		},
		Spec: virtuslabv1alpha1.JenkinsSpec{
			Master: virtuslabv1alpha1.JenkinsMaster{
				Image:       "jenkins/jenkins",
				Annotations: map[string]string{"test": "label"},
			},
			//TODO(bantoniak) add seed job with private key
			SeedJobs: []virtuslabv1alpha1.SeedJob{
				{
					ID:               "jenkins-operator",
					Targets:          "cicd/jobs/*.jenkins",
					Description:      "Jenkins Operator repository",
					RepositoryBranch: "master",
					RepositoryURL:    "https://github.com/VirtusLab/jenkins-operator.git",
				},
			},
		},
	}

	t.Logf("Jenkins CR %+v", *jenkins)
	if err := framework.Global.Client.Create(context.TODO(), jenkins, nil); err != nil {
		t.Fatal(err)
	}

	return jenkins
}

func verifyJenkinsAPIConnection(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) *gojenkins.Jenkins {
	client, err := createJenkinsAPIClient(jenkins)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("I can establish connection to Jenkins API")
	return client
}

func restartJenkinsMasterPod(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) {
	t.Log("Restarting Jenkins master pod")
	jenkinsPod := getJenkinsMasterPod(t, jenkins)
	err := framework.Global.Client.Delete(context.TODO(), jenkinsPod)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Jenkins master pod has been restarted")
}
