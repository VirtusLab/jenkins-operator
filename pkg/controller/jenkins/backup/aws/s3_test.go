package aws

import (
	"context"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestAmazonS3Backup_IsConfigurationValidForBasePhase(t *testing.T) {
	tests := []struct {
		name    string
		jenkins virtuslabv1alpha1.Jenkins
		want    bool
	}{
		{
			name: "happy",
			jenkins: virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					BackupAmazonS3: virtuslabv1alpha1.JenkinsBackupAmazonS3{
						BucketName: "some-value",
						BucketPath: "some-value",
						Region:     "some-value",
					},
				},
			},
			want: true,
		},
		{
			name: "fail, no bucket name",
			jenkins: virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					BackupAmazonS3: virtuslabv1alpha1.JenkinsBackupAmazonS3{
						BucketName: "",
						BucketPath: "some-value",
						Region:     "some-value",
					},
				},
			},
			want: false,
		},
		{
			name: "fail, no bucket path",
			jenkins: virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					BackupAmazonS3: virtuslabv1alpha1.JenkinsBackupAmazonS3{
						BucketName: "some-value",
						BucketPath: "",
						Region:     "some-value",
					},
				},
			},
			want: false,
		},
		{
			name: "fail, no region",
			jenkins: virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					BackupAmazonS3: virtuslabv1alpha1.JenkinsBackupAmazonS3{
						BucketName: "some-value",
						BucketPath: "some-value",
						Region:     "",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AmazonS3Backup{}
			got := r.IsConfigurationValidForBasePhase(tt.jenkins, logf.ZapLogger(false))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAmazonS3Backup_IsConfigurationValidForUserPhase(t *testing.T) {
	tests := []struct {
		name    string
		jenkins *virtuslabv1alpha1.Jenkins
		secret  *corev1.Secret
		want    bool
		wantErr bool
	}{
		{
			name: "happy",
			jenkins: &virtuslabv1alpha1.Jenkins{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-cr-name"},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-operator-backup-credentials-jenkins-cr-name"},
				Data: map[string][]byte{
					constants.BackupAmazonS3SecretSecretKey: []byte("some-value"),
					constants.BackupAmazonS3SecretAccessKey: []byte("some-value"),
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "fail, no secret",
			jenkins: &virtuslabv1alpha1.Jenkins{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-cr-name"},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "fail, no secret key in secret",
			jenkins: &virtuslabv1alpha1.Jenkins{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-cr-name"},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-operator-backup-credentials-jenkins-cr-name"},
				Data: map[string][]byte{
					constants.BackupAmazonS3SecretSecretKey: []byte(""),
					constants.BackupAmazonS3SecretAccessKey: []byte("some-value"),
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "fail, no access key in secret",
			jenkins: &virtuslabv1alpha1.Jenkins{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-cr-name"},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-operator-backup-credentials-jenkins-cr-name"},
				Data: map[string][]byte{
					constants.BackupAmazonS3SecretSecretKey: []byte("some-value"),
					constants.BackupAmazonS3SecretAccessKey: []byte(""),
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewFakeClient()
			logger := logf.ZapLogger(false)
			b := &AmazonS3Backup{}
			if tt.secret != nil {
				e := k8sClient.Create(context.TODO(), tt.secret)
				assert.NoError(t, e)
			}
			got, err := b.IsConfigurationValidForUserPhase(k8sClient, *tt.jenkins, logger)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
