package user

import (
	"context"
	"fmt"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var fakePrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEArK4ld6i2iqW6L3jaTZaKD/v7PjDn+Ik9MXp+kvLcUw/+wEGm
285UwqLnDDlBhSi9nDgJ+m1XU87VCpz/DXW23R/CQcMX2xunib4wWLQqoR3CWbk3
SwiLd8TWAvXkxdXm8fDOGAZbYK2alMV+M+9E2OpZsBUCxmb/3FAofF6JccKoJOH8
UveRNSOx7IXPKtHFiypBhWM4l6ZjgJKm+DRIEhyvoC+pHzcum2ZEPOv+ZJDy5jXK
ZHcNQXVnAZtCcojcjVUBw2rZms+fQ6Volv2JT71Gpykzx/rChhwNwxdAEwjLjKjL
nBWEh/WxsS3NbM7zb4B2XGMCeWVeb/niUwpy+wIDAQABAoIBAQCjGkJNidARmYQI
/u/DxWNWwb2H+o3BFW/1YixYBIjS9BK96cT/bR5mUZRG2XXnnpmqCsxx/AE2KfDU
e4H1ZrB4oFzN3MaVsMNIuZnUzyhM0l0WfnmZp9KEKCm01ilmLCpdcARacPaylIej
6f7QcznmYUShqtbaK8OUhyoWfvz3s0VLkpBlqm63uPtjAx6sAl399THxHVwbYgYy
TxPY8wdjOvNzQJ7ColUh05Zq6TsCGGFUFg7v4to/AXtDhcTMVONlapP+XxekRx8P
98BepIgzgvQhWak8gm+cKQYANk14Q8BDzUCDplYuIZVvKl+/ZHltjHGjrqxDrcDA
0U7REgtxAoGBAN+LAEf2o14ffs/ebVSxiv7LnuAxFh2L6i7RqtehpSf7BnYC65vB
6TMsc/0/KFkD5Az7nrJmA7HmM8J/NI2ks0Mbft+0XCRFx/zfU6pOvPinRKp/8Vtm
aUmNzhz8UMaQ1JXOvBOqvXKWYrN1WPha1+/BnUQrpTdhGxAoAh1FW4eHAoGBAMXA
mXTN5X8+mp9KW2bIpFsjrZ+EyhxO6a6oBMZY54rceeOzf5RcXY7EOiTrnmr+lQvp
fAKBeX5V8G96nSEIDmPhKGZ1C1vEP6hRWahJo1XkN5E1j6hRHCu3DQLtL2lxlyfG
Fx11fysgmLoPVVytLAEQwt4WxMp7OsM1NWqB+u3tAoGBAILUg3Gas7pejIV0FGDB
GCxPV8i2cc8RGBoWs/pHrLVdgUaIJwSd1LISjj/lOuP+FvZSPWsDsZ3osNpgQI21
mwTnjrW2hUblYEprGjhOpOKSYum2v7dSlMRrrfng4hWUphaXTBPmlcH+qf2F7HBO
GptDoZtIQAXNW111TOd8tDj5AoGAC1PO9nvcy38giENQHQEdOQNALMUEdr6mcBS7
wUjSaofai4p6olrwGP9wfTDp8CMJEpebPOGBvhTaIuiZG41ElcAN+mB1+Bmzs8aF
JjihnIfoDu9MfU24GWDw49wGPTn+eI7GQC+8yxGg7fd24kohHSaCowoW16pbYVco
6iLr5rkCgYBt0bcYJ3AOTH0UXS8kvJvnyce/RBIAMoUABwvdkZt9r5B4UzsoLq5e
WrrU6fSRsE6lSsBd83pOAQ46tv+vntQ+0EihD9/0INhkQM99lBw1TFdFTgGSAs1e
ns4JGP6f5uIuwqu/nbqPqMyDovjkGbX2znuGBcvki90Pi97XL7MMWw==
-----END RSA PRIVATE KEY-----
`

var fakeInvalidPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEArK4ld6i2iqW6L3jaTZaKD/v7PjDn+Ik9MXp+kvLcUw/+wEGm
285UwqLnDDlBhSi9nDgJ+m1XU87VCpz/DXW23R/CQcMX2xunib4wWLQqoR3CWbk3
SwiLd8TWAvXkxdXm8fDOGAZbYK2alMV+M+9E2OpZsBUCxmb/3FAofF6JccKoJOH8
`

func TestValidateSeedJobs(t *testing.T) {
	data := []struct {
		description    string
		jenkins        *virtuslabv1alpha1.Jenkins
		secret         *corev1.Secret
		expectedResult bool
	}{
		{
			description: "Valid with public repository and without private key",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
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
			},
			expectedResult: true,
		},
		{
			description: "Invalid without id",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					SeedJobs: []virtuslabv1alpha1.SeedJob{
						{
							Targets:          "cicd/jobs/*.jenkins",
							Description:      "Jenkins Operator e2e tests repository",
							RepositoryBranch: "master",
							RepositoryURL:    "https://github.com/VirtusLab/jenkins-operator-e2e.git",
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			description: "Valid with private key and secret",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					SeedJobs: []virtuslabv1alpha1.SeedJob{
						{
							ID:               "jenkins-operator-e2e",
							Targets:          "cicd/jobs/*.jenkins",
							Description:      "Jenkins Operator e2e tests repository",
							RepositoryBranch: "master",
							RepositoryURL:    "https://github.com/VirtusLab/jenkins-operator-e2e.git",
							PrivateKey: virtuslabv1alpha1.PrivateKey{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "deploy-keys",
									},
									Key: "jenkins-operator-e2e",
								},
							},
						},
					},
				},
			},
			secret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deploy-keys",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"jenkins-operator-e2e": []byte(fakePrivateKey),
				},
			},
			expectedResult: true,
		},
		{
			description: "Invalid private key in secret",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					SeedJobs: []virtuslabv1alpha1.SeedJob{
						{
							ID:               "jenkins-operator-e2e",
							Targets:          "cicd/jobs/*.jenkins",
							Description:      "Jenkins Operator e2e tests repository",
							RepositoryBranch: "master",
							RepositoryURL:    "https://github.com/VirtusLab/jenkins-operator-e2e.git",
							PrivateKey: virtuslabv1alpha1.PrivateKey{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "deploy-keys",
									},
									Key: "jenkins-operator-e2e",
								},
							},
						},
					},
				},
			},
			secret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deploy-keys",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"jenkins-operator-e2e": []byte(fakeInvalidPrivateKey),
				},
			},
			expectedResult: false,
		},
		{
			description: "Invalid with PrivateKey and empty Secret data",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					SeedJobs: []virtuslabv1alpha1.SeedJob{
						{
							ID:               "jenkins-operator-e2e",
							Targets:          "cicd/jobs/*.jenkins",
							Description:      "Jenkins Operator e2e tests repository",
							RepositoryBranch: "master",
							RepositoryURL:    "https://github.com/VirtusLab/jenkins-operator-e2e.git",
							PrivateKey: virtuslabv1alpha1.PrivateKey{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "deploy-keys",
									},
									Key: "jenkins-operator-e2e",
								},
							},
						},
					},
				},
			},
			secret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deploy-keys",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"jenkins-operator-e2e": []byte(""),
				},
			},
			expectedResult: false,
		},
		{
			description: "Invalid with ssh RepositoryURL and empty PrivateKey",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					SeedJobs: []virtuslabv1alpha1.SeedJob{
						{
							ID:               "jenkins-operator-e2e",
							Targets:          "cicd/jobs/*.jenkins",
							Description:      "Jenkins Operator e2e tests repository",
							RepositoryBranch: "master",
							RepositoryURL:    "git@github.com:VirtusLab/jenkins-operator.git",
						},
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, testingData := range data {
		t.Run(fmt.Sprintf("Testing '%s'", testingData.description), func(t *testing.T) {
			fakeClient := fake.NewFakeClient()
			if testingData.secret != nil {
				err := fakeClient.Create(context.TODO(), testingData.secret)
				assert.NoError(t, err)
			}
			userReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false), nil)
			result, err := userReconcileLoop.validateSeedJobs(testingData.jenkins)
			assert.NoError(t, err)
			assert.Equal(t, testingData.expectedResult, result)
		})
	}
}

func TestReconcileUserConfiguration_verifyBackupAmazonS3(t *testing.T) {
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
			r := &ReconcileUserConfiguration{
				k8sClient:     fake.NewFakeClient(),
				jenkinsClient: nil,
				logger:        logf.ZapLogger(false),
				jenkins:       tt.jenkins,
			}
			if tt.secret != nil {
				e := r.k8sClient.Create(context.TODO(), tt.secret)
				assert.NoError(t, e)
			}
			got, err := r.verifyBackupAmazonS3()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
