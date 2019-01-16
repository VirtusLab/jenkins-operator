package e2e

import (
	"context"
	"reflect"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugins"

	"github.com/bndr/gojenkins"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBaseConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	jenkins := createJenkinsCR(t, namespace)
	createDefaultLimitsForContainersInNamespace(t, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)

	verifyJenkinsMasterPodAttributes(t, jenkins)
	jenkinsClient := verifyJenkinsAPIConnection(t, jenkins)
	verifyBasePlugins(t, jenkinsClient)
}

func createDefaultLimitsForContainersInNamespace(t *testing.T, namespace string) {
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e",
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					DefaultRequest: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Default: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
		},
	}

	t.Logf("LimitRange %+v", *limitRange)
	if err := framework.Global.Client.Create(context.TODO(), limitRange, nil); err != nil {
		t.Fatal(err)
	}
}

func verifyJenkinsMasterPodAttributes(t *testing.T, jenkins *virtuslabv1alpha1.Jenkins) {
	jenkinsPod := getJenkinsMasterPod(t, jenkins)
	jenkins = getJenkins(t, jenkins.Namespace, jenkins.Name)

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

func verifyBasePlugins(t *testing.T, jenkinsClient *gojenkins.Jenkins) {
	installedPlugins, err := jenkinsClient.GetPlugins(1)
	if err != nil {
		t.Fatal(err)
	}

	for rootPluginName, p := range plugins.BasePluginsMap {
		rootPlugin, err := plugins.New(rootPluginName)
		if err != nil {
			t.Fatal(err)
		}
		if found, ok := isPluginValid(installedPlugins, *rootPlugin); !ok {
			t.Fatalf("Invalid plugin '%s', actual '%+v'", rootPlugin, found)
		}
		for _, requiredPlugin := range p {
			if found, ok := isPluginValid(installedPlugins, requiredPlugin); !ok {
				t.Fatalf("Invalid plugin '%s', actual '%+v'", requiredPlugin, found)
			}
		}
	}

	t.Log("Base plugins have been installed")
}

func isPluginValid(plugins *gojenkins.Plugins, requiredPlugin plugins.Plugin) (*gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return p, false
	}

	if !p.Active || !p.Enabled || p.Deleted {
		return p, false
	}

	return p, requiredPlugin.Version == p.Version
}
