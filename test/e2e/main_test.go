package e2e

import (
	"flag"
	"testing"

	"github.com/VirtusLab/jenkins-operator/pkg/apis"
	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	f "github.com/operator-framework/operator-sdk/pkg/test"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	jenkinsOperatorDeploymentName            = constants.OperatorName
	amazonS3BackupConfigurationParameterName = "s3BackupConfig"
)

var (
	amazonS3BackupConfigurationFile *string
)

func TestMain(m *testing.M) {
	amazonS3BackupConfigurationFile = flag.String(amazonS3BackupConfigurationParameterName, "", "path to AWS S3 backup config")
	f.MainEntry(m)
}

func setupTest(t *testing.T) (string, *framework.TestCtx) {
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(nil)
	if err != nil {
		t.Fatalf("could not initialize cluster resources: %v", err)
	}

	jenkinsServiceList := &virtuslabv1alpha1.JenkinsList{
		TypeMeta: metav1.TypeMeta{
			Kind:       virtuslabv1alpha1.Kind,
			APIVersion: virtuslabv1alpha1.SchemeGroupVersion.String(),
		},
	}
	err = framework.AddToFrameworkScheme(apis.AddToScheme, jenkinsServiceList)
	if err != nil {
		t.Fatalf("could not add scheme to framework scheme: %v", err)
	}

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("could not get namespace: %v", err)
	}
	t.Logf("Test namespace '%s'", namespace)

	// wait for jenkins-operator to be ready
	err = e2eutil.WaitForDeployment(t, framework.Global.KubeClient, namespace, jenkinsOperatorDeploymentName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	return namespace, ctx
}
