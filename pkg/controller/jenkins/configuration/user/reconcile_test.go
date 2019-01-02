package user

import (
	"context"
	"os"
	"testing"
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/jobs"

	"github.com/bndr/gojenkins"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestMain(m *testing.M) {
	virtuslabv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
	os.Exit(m.Run())
}

func TestReconcileUserConfiguration(t *testing.T) {
	// given
	logger := logf.ZapLogger(false)
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	jenkinsClient := client.NewMockJenkins(ctrl)
	fakeClient := fake.NewFakeClient()

	jenkins := jenkinsCustomResource()
	err := fakeClient.Create(ctx, jenkins)
	assert.NoError(t, err)

	reconcile := New(fakeClient, jenkinsClient, logger, jenkins)

	// first run - should ensure seed jobs(configure seed jobs, run build) and requeue reconciliation loop
	jenkinsClient.
		EXPECT().
		CreateOrUpdateJob(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	jenkinsClient.
		EXPECT().
		BuildJob(gomock.Any(), gomock.Any()).
		Return(int64(1), nil)

	result, err := reconcile.Reconcile()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Requeue)
	assert.Equal(t, result.RequeueAfter, time.Second*10)

	// second run - should ensure seed jobs(configure/update seed jobs, finish build) and requeue reconciliation loop
	jenkinsClient.
		EXPECT().
		GetBuild(gomock.Any(), gomock.Any()).
		Return(&gojenkins.Build{
			Raw: &gojenkins.BuildResponse{
				Result: jobs.SuccessStatus,
			},
		}, nil)

	jenkinsClient.
		EXPECT().
		CreateOrUpdateJob(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	result, err = reconcile.Reconcile()
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func jenkinsCustomResource() *virtuslabv1alpha1.Jenkins {
	return &virtuslabv1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins",
			Namespace: "default",
		},
		Spec: virtuslabv1alpha1.JenkinsSpec{
			Master: virtuslabv1alpha1.JenkinsMaster{
				Image:       "jenkins/jenkins",
				Annotations: map[string]string{"test": "label"},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("500Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
			SeedJobs: []virtuslabv1alpha1.SeedJob{
				{
					ID:               "jenkins-operator-e2e",
					Targets:          "cicd/jobs/*.jenkins",
					Description:      "Jenkins Operator e2e tests repository",
					RepositoryBranch: "master",
					RepositoryURL:    "https://github.com/VirtusLab/jenkins-operator-e2e.git",
				},
			},
		},
	}
}
