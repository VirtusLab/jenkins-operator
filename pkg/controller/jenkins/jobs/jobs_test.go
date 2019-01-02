package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/bndr/gojenkins"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestMain(m *testing.M) {
	virtuslabv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
	os.Exit(m.Run())
}

func TestSuccessEnsureJob(t *testing.T) {
	// given
	logger := logf.ZapLogger(false)
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	buildName := "Test Job"
	buildNumber := int64(1)
	jenkinsClient := client.NewMockJenkins(ctrl)
	fakeClient := fake.NewFakeClient()

	hash := sha256.New()
	hash.Write([]byte(buildName))
	encodedHash := base64.URLEncoding.EncodeToString(hash.Sum(nil))

	// when
	jenkins := jenkinsCustomResource()
	err := fakeClient.Create(ctx, jenkins)
	assert.NoError(t, err)

	jobs := New(jenkinsClient, fakeClient, logger)

	// first run - build should be scheduled and status updated
	jenkinsClient.
		EXPECT().
		BuildJob(buildName, gomock.Any()).
		Return(buildNumber, nil)

	done, err := jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
	assert.NoError(t, err)
	assert.False(t, done)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
	assert.NoError(t, err)

	assert.NotEmpty(t, jenkins.Status.Builds)
	assert.Equal(t, len(jenkins.Status.Builds), 1)

	build := jenkins.Status.Builds[0]
	assert.Equal(t, build.Name, buildName)
	assert.Equal(t, build.Hash, encodedHash)
	assert.Equal(t, build.Number, buildNumber)
	assert.Equal(t, build.Status, RunningStatus)
	assert.Equal(t, build.Retires, 0)
	assert.NotNil(t, build.CreateTime)
	assert.NotNil(t, build.LastUpdateTime)

	// second run - build should be success and status updated
	jenkinsClient.
		EXPECT().
		GetBuild(buildName, buildNumber).
		Return(&gojenkins.Build{
			Raw: &gojenkins.BuildResponse{
				Result: SuccessStatus,
			},
		}, nil)

	done, err = jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
	assert.NoError(t, err)
	assert.True(t, done)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
	assert.NoError(t, err)

	assert.NotEmpty(t, jenkins.Status.Builds)
	assert.Equal(t, len(jenkins.Status.Builds), 1)

	build = jenkins.Status.Builds[0]
	assert.Equal(t, build.Name, buildName)
	assert.Equal(t, build.Hash, encodedHash)
	assert.Equal(t, build.Number, buildNumber)
	assert.Equal(t, build.Status, SuccessStatus)
	assert.Equal(t, build.Retires, 0)
	assert.NotNil(t, build.CreateTime)
	assert.NotNil(t, build.LastUpdateTime)
}

func TestEnsureJobWithFailedBuild(t *testing.T) {
	// given
	logger := logf.ZapLogger(false)
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	buildName := "Test Job"
	buildNumber := int64(1)
	jenkinsClient := client.NewMockJenkins(ctrl)
	fakeClient := fake.NewFakeClient()

	hash := sha256.New()
	hash.Write([]byte(buildName))
	encodedHash := base64.URLEncoding.EncodeToString(hash.Sum(nil))

	// when
	jenkins := jenkinsCustomResource()
	err := fakeClient.Create(ctx, jenkins)
	assert.NoError(t, err)

	jobs := New(jenkinsClient, fakeClient, logger)

	// first run - build should be scheduled and status updated
	jenkinsClient.
		EXPECT().
		BuildJob(buildName, gomock.Any()).
		Return(buildNumber, nil)

	done, err := jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
	assert.NoError(t, err)
	assert.False(t, done)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
	assert.NoError(t, err)

	assert.NotEmpty(t, jenkins.Status.Builds)
	assert.Equal(t, len(jenkins.Status.Builds), 1)

	build := jenkins.Status.Builds[0]
	assert.Equal(t, build.Name, buildName)
	assert.Equal(t, build.Hash, encodedHash)
	assert.Equal(t, build.Number, buildNumber)
	assert.Equal(t, build.Status, RunningStatus)
	assert.Equal(t, build.Retires, 0)
	assert.NotNil(t, build.CreateTime)
	assert.NotNil(t, build.LastUpdateTime)

	// second run - build should be failure and status updated
	jenkinsClient.
		EXPECT().
		GetBuild(buildName, buildNumber).
		Return(&gojenkins.Build{
			Raw: &gojenkins.BuildResponse{
				Result: FailureStatus,
			},
		}, nil)

	done, err = jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
	assert.NoError(t, err)
	assert.False(t, done)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
	assert.NoError(t, err)

	assert.NotEmpty(t, jenkins.Status.Builds)
	assert.Equal(t, len(jenkins.Status.Builds), 1)

	build = jenkins.Status.Builds[0]
	assert.Equal(t, build.Name, buildName)
	assert.Equal(t, build.Hash, encodedHash)
	assert.Equal(t, build.Number, buildNumber)
	assert.Equal(t, build.Status, FailureStatus)
	assert.Equal(t, build.Retires, 0)
	assert.NotNil(t, build.CreateTime)
	assert.NotNil(t, build.LastUpdateTime)

	// third run - build should be rescheduled and status updated
	jenkinsClient.
		EXPECT().
		BuildJob(buildName, gomock.Any()).
		Return(buildNumber+1, nil)

	done, err = jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
	assert.EqualError(t, err, ErrorBuildFailed.Error())
	assert.False(t, done)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
	assert.NoError(t, err)

	assert.NotEmpty(t, jenkins.Status.Builds)
	assert.Equal(t, len(jenkins.Status.Builds), 1)

	build = jenkins.Status.Builds[0]
	assert.Equal(t, build.Name, buildName)
	assert.Equal(t, build.Hash, encodedHash)
	assert.Equal(t, build.Number, buildNumber+1)
	assert.Equal(t, build.Status, RunningStatus)
	assert.Equal(t, build.Retires, 1)
	assert.NotNil(t, build.CreateTime)
	assert.NotNil(t, build.LastUpdateTime)

	// fourth run - build should be success and status updated
	jenkinsClient.
		EXPECT().
		GetBuild(buildName, buildNumber+1).
		Return(&gojenkins.Build{
			Raw: &gojenkins.BuildResponse{
				Result: SuccessStatus,
			},
		}, nil)

	done, err = jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
	assert.NoError(t, err)
	assert.True(t, done)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
	assert.NoError(t, err)

	assert.NotEmpty(t, jenkins.Status.Builds)
	assert.Equal(t, len(jenkins.Status.Builds), 1)

	build = jenkins.Status.Builds[0]
	assert.Equal(t, build.Name, buildName)
	assert.Equal(t, build.Hash, encodedHash)
	assert.Equal(t, build.Number, buildNumber+1)
	assert.Equal(t, build.Status, SuccessStatus)
	assert.Equal(t, build.Retires, 1)
	assert.NotNil(t, build.CreateTime)
	assert.NotNil(t, build.LastUpdateTime)
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
