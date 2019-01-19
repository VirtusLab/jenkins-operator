package base

import (
	"context"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugins"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestReconcileJenkinsBaseConfiguration_ensurePluginsRequiredByAllBackupProviders(t *testing.T) {
	tests := []struct {
		name            string
		jenkins         *virtuslabv1alpha1.Jenkins
		requiredPlugins map[string][]plugins.Plugin
		want            reconcile.Result
		wantErr         bool
	}{
		{
			name: "happy, no required plugins",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					Master: virtuslabv1alpha1.JenkinsMaster{
						Plugins: map[string][]string{
							"first-plugin:0.0.1": {"second-plugin:0.0.1"},
						},
					},
				},
			},
			want:    reconcile.Result{Requeue: false},
			wantErr: false,
		},
		{
			name: "happy, required plugins are set",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					Master: virtuslabv1alpha1.JenkinsMaster{
						Plugins: map[string][]string{
							"first-plugin:0.0.1": {"second-plugin:0.0.1"},
						},
					},
				},
			},
			requiredPlugins: map[string][]plugins.Plugin{
				"first-plugin:0.0.1": {plugins.Must(plugins.New("second-plugin:0.0.1"))},
			},
			want:    reconcile.Result{Requeue: false},
			wantErr: false,
		},
		{
			name: "happy, jenkins CR must be updated",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					Master: virtuslabv1alpha1.JenkinsMaster{
						Plugins: map[string][]string{
							"first-plugin:0.0.1": {"second-plugin:0.0.1"},
						},
					},
				},
			},
			requiredPlugins: map[string][]plugins.Plugin{
				"first-plugin:0.0.1": {plugins.Must(plugins.New("second-plugin:0.0.1"))},
				"third-plugin:0.0.1": {},
			},
			want:    reconcile.Result{Requeue: true},
			wantErr: false,
		},
		{
			name: "happy, jenkins CR must be updated",
			jenkins: &virtuslabv1alpha1.Jenkins{
				Spec: virtuslabv1alpha1.JenkinsSpec{
					Master: virtuslabv1alpha1.JenkinsMaster{
						Plugins: map[string][]string{
							"first-plugin:0.0.1": {"second-plugin:0.0.1"},
						},
					},
				},
			},
			requiredPlugins: map[string][]plugins.Plugin{
				"first-plugin:0.0.1": {plugins.Must(plugins.New("second-plugin:0.0.1"))},
				"third-plugin:0.0.1": {plugins.Must(plugins.New("fourth-plugin:0.0.1"))},
			},
			want:    reconcile.Result{Requeue: true},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := virtuslabv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)
			r := &ReconcileJenkinsBaseConfiguration{
				k8sClient: fake.NewFakeClient(),
				scheme:    nil,
				logger:    logf.ZapLogger(false),
				jenkins:   tt.jenkins,
				local:     false,
				minikube:  false,
			}
			err = r.k8sClient.Create(context.TODO(), tt.jenkins)
			assert.NoError(t, err)
			got, err := r.ensurePluginsRequiredByAllBackupProviders(tt.requiredPlugins)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
