package e2e

import (
	"reflect"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"

	"github.com/bndr/gojenkins"
)

func TestBaseConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	jenkins := createJenkinsCR(t, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)

	verifyJenkinsMasterPodAttributes(t, jenkins)
	verifyJenkinsAPIConnection(t, jenkins)
}

func verifyJenkinsAPIConnection(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) *gojenkins.Jenkins {
	client, err := createJenkinsAPIClient(jenkins)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("I can establish connection to Jenkins API")
	return client
}

func verifyJenkinsMasterPodAttributes(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) {
	jenkinsPod := getJenkinsMasterPod(t, jenkins)

	for key, value := range jenkins.Spec.Master.Annotations {
		if jenkinsPod.ObjectMeta.Annotations[key] != value {
			t.Fatalf("Invalid Jenkins pod annotation expected '%+v', actual '%+v'", jenkins.Spec.Master.Annotations, jenkinsPod.ObjectMeta.Annotations)
		}
	}

	if jenkinsPod.Spec.Containers[0].Image != jenkins.Spec.Master.Image {
		t.Fatalf("Invalid jenkins pod image expected '%s', actual '%s'", jenkins.Spec.Master.Image, jenkinsPod.Spec.Containers[0].Image)
	}

	if !reflect.DeepEqual(jenkinsPod.Spec.Containers[0].Resources, jenkins.Spec.Master.Resources) {
		t.Fatalf("Invalid jenkins pod continer resources expected '%+v', actual '%+v'", jenkins.Spec.Master.Resources, jenkinsPod.Spec.Containers[0].Resources)
	}

	t.Log("Jenkins pod attributes are valid")
}
