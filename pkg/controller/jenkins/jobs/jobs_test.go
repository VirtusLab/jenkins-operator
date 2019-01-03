package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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
	ctx := context.TODO()
	logger := logf.ZapLogger(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	buildName := "Test Job"
	hash := sha256.New()
	hash.Write([]byte(buildName))
	encodedHash := base64.URLEncoding.EncodeToString(hash.Sum(nil))

	// when
	jenkins := jenkinsCustomResource()
	fakeClient := fake.NewFakeClient()
	err := fakeClient.Create(ctx, jenkins)
	assert.NoError(t, err)

	for reconcileAttempt := 1; reconcileAttempt <= 2; reconcileAttempt++ {
		logger.Info(fmt.Sprintf("Reconcile attempt #%d", reconcileAttempt))
		buildNumber := int64(1)
		jenkinsClient := client.NewMockJenkins(ctrl)
		jobs := New(jenkinsClient, fakeClient, logger)

		jenkinsClient.
			EXPECT().
			GetJob(buildName).
			Return(&gojenkins.Job{
				Raw: &gojenkins.JobResponse{
					NextBuildNumber: buildNumber,
				},
			}, nil).AnyTimes()

		jenkinsClient.
			EXPECT().
			BuildJob(buildName, gomock.Any()).
			Return(int64(0), nil).AnyTimes()

		jenkinsClient.
			EXPECT().
			GetBuild(buildName, buildNumber).
			Return(&gojenkins.Build{
				Raw: &gojenkins.BuildResponse{
					Result: SuccessStatus,
				},
			}, nil).AnyTimes()

		done, err := jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
		assert.NoError(t, err)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		assert.NoError(t, err)

		assert.NotEmpty(t, jenkins.Status.Builds)
		assert.Equal(t, len(jenkins.Status.Builds), 1)

		build := jenkins.Status.Builds[0]
		assert.Equal(t, build.Name, buildName)
		assert.Equal(t, build.Hash, encodedHash)
		assert.Equal(t, build.Number, buildNumber)
		assert.Equal(t, build.Retires, 0)
		assert.NotNil(t, build.CreateTime)
		assert.NotNil(t, build.LastUpdateTime)

		// first run - build should be scheduled and status updated
		if reconcileAttempt == 1 {
			assert.False(t, done)
			assert.Equal(t, build.Status, RunningStatus)
		}

		// second run -job should be success and status updated
		if reconcileAttempt == 2 {
			assert.True(t, done)
			assert.Equal(t, build.Status, SuccessStatus)
		}
	}
}

func TestEnsureJobWithFailedBuild(t *testing.T) {
	// given
	ctx := context.TODO()
	logger := logf.ZapLogger(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	buildName := "Test Job"
	hash := sha256.New()
	hash.Write([]byte(buildName))
	encodedHash := base64.URLEncoding.EncodeToString(hash.Sum(nil))

	// when
	jenkins := jenkinsCustomResource()
	fakeClient := fake.NewFakeClient()
	err := fakeClient.Create(ctx, jenkins)
	assert.NoError(t, err)

	for reconcileAttempt := 1; reconcileAttempt <= 4; reconcileAttempt++ {
		logger.Info(fmt.Sprintf("Reconcile attempt #%d", reconcileAttempt))
		jenkinsClient := client.NewMockJenkins(ctrl)
		jobs := New(jenkinsClient, fakeClient, logger)

		// first run - build should be scheduled and status updated
		if reconcileAttempt == 1 {
			jenkinsClient.
				EXPECT().
				GetJob(buildName).
				Return(&gojenkins.Job{
					Raw: &gojenkins.JobResponse{
						NextBuildNumber: int64(1),
					},
				}, nil)

			jenkinsClient.
				EXPECT().
				BuildJob(buildName, gomock.Any()).
				Return(int64(0), nil)
		}

		// second run - build should be failure and status updated
		if reconcileAttempt == 2 {
			jenkinsClient.
				EXPECT().
				GetBuild(buildName, int64(1)).
				Return(&gojenkins.Build{
					Raw: &gojenkins.BuildResponse{
						Result: FailureStatus,
					},
				}, nil)
		}

		// third run - build should be rescheduled and status updated
		if reconcileAttempt == 3 {
			jenkinsClient.
				EXPECT().
				GetJob(buildName).
				Return(&gojenkins.Job{
					Raw: &gojenkins.JobResponse{
						NextBuildNumber: int64(2),
					},
				}, nil)

			jenkinsClient.
				EXPECT().
				BuildJob(buildName, gomock.Any()).
				Return(int64(0), nil)
		}

		// fourth run - build should be success and status updated
		if reconcileAttempt == 4 {
			jenkinsClient.
				EXPECT().
				GetBuild(buildName, int64(2)).
				Return(&gojenkins.Build{
					Raw: &gojenkins.BuildResponse{
						Result: SuccessStatus,
					},
				}, nil)
		}

		done, errEnsureBuildJob := jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
		assert.NoError(t, err)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		assert.NoError(t, err)

		assert.NotEmpty(t, jenkins.Status.Builds)
		assert.Equal(t, len(jenkins.Status.Builds), 1)

		build := jenkins.Status.Builds[0]
		assert.Equal(t, build.Name, buildName)
		assert.Equal(t, build.Hash, encodedHash)

		assert.NotNil(t, build.CreateTime)
		assert.NotNil(t, build.LastUpdateTime)

		// first run - build should be scheduled and status updated
		if reconcileAttempt == 1 {
			assert.NoError(t, errEnsureBuildJob)
			assert.False(t, done)
			assert.Equal(t, build.Number, int64(1))
			assert.Equal(t, build.Status, RunningStatus)
		}

		// second run - build should be failure and status updated
		if reconcileAttempt == 2 {
			assert.Error(t, errEnsureBuildJob)
			assert.False(t, done)
			assert.Equal(t, build.Number, int64(1))
			assert.Equal(t, build.Status, FailureStatus)
		}

		// third run - build should be rescheduled and status updated
		if reconcileAttempt == 3 {
			assert.NoError(t, errEnsureBuildJob)
			assert.False(t, done)
			assert.Equal(t, build.Number, int64(2))
			assert.Equal(t, build.Status, RunningStatus)
		}

		// fourth run - build should be success and status updated
		if reconcileAttempt == 4 {
			assert.NoError(t, errEnsureBuildJob)
			assert.True(t, done)
			assert.Equal(t, build.Number, int64(2))
			assert.Equal(t, build.Status, SuccessStatus)
		}
	}
}

func TestEnsureJobFailedWithMaxRetries(t *testing.T) {
	// given
	ctx := context.TODO()
	logger := logf.ZapLogger(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	buildName := "Test Job"
	hash := sha256.New()
	hash.Write([]byte(buildName))
	encodedHash := base64.URLEncoding.EncodeToString(hash.Sum(nil))

	// when
	jenkins := jenkinsCustomResource()
	fakeClient := fake.NewFakeClient()
	err := fakeClient.Create(ctx, jenkins)
	assert.NoError(t, err)

	BuildRetires = 1 // override max build retries
	for reconcileAttempt := 1; reconcileAttempt <= 5; reconcileAttempt++ {
		logger.Info(fmt.Sprintf("Reconcile attempt #%d", reconcileAttempt))
		jenkinsClient := client.NewMockJenkins(ctrl)
		jobs := New(jenkinsClient, fakeClient, logger)

		// first run - build should be scheduled and status updated
		if reconcileAttempt == 1 {
			jenkinsClient.
				EXPECT().
				GetJob(buildName).
				Return(&gojenkins.Job{
					Raw: &gojenkins.JobResponse{
						NextBuildNumber: int64(1),
					},
				}, nil)

			jenkinsClient.
				EXPECT().
				BuildJob(buildName, gomock.Any()).
				Return(int64(0), nil)
		}

		// second run - build should be failure and status updated
		if reconcileAttempt == 2 {
			jenkinsClient.
				EXPECT().
				GetBuild(buildName, int64(1)).
				Return(&gojenkins.Build{
					Raw: &gojenkins.BuildResponse{
						Result: FailureStatus,
					},
				}, nil)
		}

		// third run - build should be rescheduled and status updated
		if reconcileAttempt == 3 {
			jenkinsClient.
				EXPECT().
				GetJob(buildName).
				Return(&gojenkins.Job{
					Raw: &gojenkins.JobResponse{
						NextBuildNumber: int64(2),
					},
				}, nil)

			jenkinsClient.
				EXPECT().
				BuildJob(buildName, gomock.Any()).
				Return(int64(0), nil)
		}

		// fourth run - build should be success and status updated
		if reconcileAttempt == 4 {
			jenkinsClient.
				EXPECT().
				GetBuild(buildName, int64(2)).
				Return(&gojenkins.Build{
					Raw: &gojenkins.BuildResponse{
						Result: FailureStatus,
					},
				}, nil)
		}

		done, errEnsureBuildJob := jobs.EnsureBuildJob(buildName, encodedHash, nil, jenkins, true)
		assert.NoError(t, err)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		assert.NoError(t, err)

		assert.NotEmpty(t, jenkins.Status.Builds)
		assert.Equal(t, len(jenkins.Status.Builds), 1)

		build := jenkins.Status.Builds[0]
		assert.Equal(t, build.Name, buildName)
		assert.Equal(t, build.Hash, encodedHash)

		assert.NotNil(t, build.CreateTime)
		assert.NotNil(t, build.LastUpdateTime)

		// first run - build should be scheduled and status updated
		if reconcileAttempt == 1 {
			assert.NoError(t, errEnsureBuildJob)
			assert.False(t, done)
			assert.Equal(t, build.Number, int64(1))
			assert.Equal(t, build.Retires, 0)
			assert.Equal(t, build.Status, RunningStatus)
		}

		// second run - build should be failure and status updated
		if reconcileAttempt == 2 {
			assert.EqualError(t, errEnsureBuildJob, ErrorBuildFailed.Error())
			assert.False(t, done)
			assert.Equal(t, build.Number, int64(1))
			assert.Equal(t, build.Retires, 0)
			assert.Equal(t, build.Status, FailureStatus)
		}

		// third run - build should be rescheduled and status updated
		if reconcileAttempt == 3 {
			assert.NoError(t, errEnsureBuildJob)
			assert.False(t, done)
			//assert.Equal(t, build.Retires, 1)
			assert.Equal(t, build.Number, int64(2))
			assert.Equal(t, build.Retires, 1)
			assert.Equal(t, build.Status, RunningStatus)
		}

		// fourth run - build should be failure and status updated
		if reconcileAttempt == 4 {
			assert.EqualError(t, errEnsureBuildJob, ErrorBuildFailed.Error())
			assert.False(t, done)
			assert.Equal(t, build.Number, int64(2))
			assert.Equal(t, build.Retires, 1)
			assert.Equal(t, build.Status, FailureStatus)
		}

		// fifth run - build should be unrecoverable failed and status updated
		if reconcileAttempt == 5 {
			assert.EqualError(t, errEnsureBuildJob, ErrorUnrecoverableBuildFailed.Error())
			assert.False(t, done)
			assert.Equal(t, build.Number, int64(2))
			assert.Equal(t, build.Retires, 1)
			assert.Equal(t, build.Status, FailureStatus)
		}
	}
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
