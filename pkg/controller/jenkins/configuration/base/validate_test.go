package base

import (
	"context"
	"fmt"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestValidatePlugins(t *testing.T) {
	data := []struct {
		plugins        map[string][]string
		expectedResult bool
	}{
		{
			plugins: map[string][]string{
				"valid-plugin-name:1.0": {
					"valid-plugin-name:1.0",
				},
			},
			expectedResult: true,
		},
		{
			plugins: map[string][]string{
				"invalid-plugin-name": {
					"invalid-plugin-name",
				},
			},
			expectedResult: false,
		},
		{
			plugins: map[string][]string{
				"valid-plugin-name:1.0": {
					"valid-plugin-name:1.0",
					"valid-plugin-name2:1.0",
				},
			},
			expectedResult: true,
		},
		{
			plugins: map[string][]string{
				"valid-plugin-name:1.0": {},
			},
			expectedResult: true,
		},
	}

	baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
		nil, false, false)

	for index, testingData := range data {
		t.Run(fmt.Sprintf("Testing %d plugins set", index), func(t *testing.T) {
			result := baseReconcileLoop.validatePlugins(testingData.plugins)
			assert.Equal(t, testingData.expectedResult, result)
		})
	}
}

func TestReconcileJenkinsBaseConfiguration_verifyBackup(t *testing.T) {
	tests := []struct {
		name    string
		jenkins *virtuslabv1alpha1.Jenkins
		secret  *corev1.Secret
		want    bool
		wantErr bool
	}{
		{
			name: "happy, no backup",
			jenkins: &virtuslabv1alpha1.Jenkins{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-cr-name"},
				Spec: virtuslabv1alpha1.JenkinsSpec{
					Backup: virtuslabv1alpha1.JenkinsBackupTypeNoBackup,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "happy",
			jenkins: &virtuslabv1alpha1.Jenkins{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-cr-name"},
				Spec: virtuslabv1alpha1.JenkinsSpec{
					Backup: virtuslabv1alpha1.JenkinsBackupTypeAmazonS3,
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-operator-backup-credentials-jenkins-cr-name"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "fail, no secret",
			jenkins: &virtuslabv1alpha1.Jenkins{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-cr-name"},
				Spec: virtuslabv1alpha1.JenkinsSpec{
					Backup: virtuslabv1alpha1.JenkinsBackupTypeAmazonS3,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "fail, empty backup type",
			jenkins: &virtuslabv1alpha1.Jenkins{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-cr-name"},
				Spec: virtuslabv1alpha1.JenkinsSpec{
					Backup: "",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-name", Name: "jenkins-operator-backup-credentials-jenkins-cr-name"},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileJenkinsBaseConfiguration{
				k8sClient: fake.NewFakeClient(),
				scheme:    nil,
				logger:    logf.ZapLogger(false),
				jenkins:   tt.jenkins,
				local:     false,
				minikube:  false,
			}
			if tt.secret != nil {
				e := r.k8sClient.Create(context.TODO(), tt.secret)
				assert.NoError(t, e)
			}
			got, err := r.verifyBackup()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
