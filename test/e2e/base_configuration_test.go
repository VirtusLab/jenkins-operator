package e2e

import (
	"reflect"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugin"

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
	jenkinsClient := verifyJenkinsAPIConnection(t, jenkins)
	verifyBasePlugins(t, jenkinsClient)
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

func verifyBasePlugins(t *testing.T, jenkinsClient *gojenkins.Jenkins) {
	allPluginsInJenkins, err := jenkinsClient.GetPlugins(1)
	if err != nil {
		t.Fatal(err)
	}

	for rootPluginName, p := range plugin.BasePluginsMap {
		rootPlugin, err := plugin.New(rootPluginName)
		if err != nil {
			t.Fatal(err)
		}
		if found, ok := isPluginValid(t, allPluginsInJenkins, *rootPlugin); !ok {
			t.Fatalf("Invalid plugin '%s', actual '%+v'", rootPlugin, found)
		}
		for _, requiredPlugin := range p {
			if found, ok := isPluginValid(t, allPluginsInJenkins, requiredPlugin); !ok {
				t.Fatalf("Invalid plugin '%s', actual '%+v'", requiredPlugin, found)
			}
		}
	}

	t.Log("Base plugins have been installed")
}

func isPluginValid(t *testing.T, plugins *gojenkins.Plugins, requiredPlugin plugin.Plugin) (*gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return p, false
	}

	if !p.Active || !p.Enabled || p.Deleted {
		return p, false
	}

	return p, requiredPlugin.Version == p.Version
}
