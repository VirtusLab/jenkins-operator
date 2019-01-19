package base

import (
	"context"
	"fmt"
	"regexp"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/backup"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugins"
	"github.com/VirtusLab/jenkins-operator/pkg/log"

	docker "github.com/docker/distribution/reference"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var (
	dockerImageRegexp = regexp.MustCompile(`^` + docker.TagRegexp.String() + `$`)
)

// Validate validates Jenkins CR Spec.master section
func (r *ReconcileJenkinsBaseConfiguration) Validate(jenkins *virtuslabv1alpha1.Jenkins) (bool, error) {
	if jenkins.Spec.Master.Image == "" {
		r.logger.V(log.VWarn).Info("Image not set")
		return false, nil
	}

	if !dockerImageRegexp.MatchString(jenkins.Spec.Master.Image) && !docker.ReferenceRegexp.MatchString(jenkins.Spec.Master.Image) {
		r.logger.V(log.VWarn).Info("Invalid image")
		return false, nil

	}

	if !r.validatePlugins(jenkins.Spec.Master.Plugins) {
		return false, nil
	}

	valid, err := r.verifyBackup()
	if !valid || err != nil {
		return valid, err
	}

	backupProvider, err := backup.GetBackupProvider(r.jenkins.Spec.Backup)
	if err != nil {
		return false, err
	}

	if !backupProvider.IsConfigurationValidForBasePhase(*r.jenkins, r.logger) {
		return false, nil
	}

	return true, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validatePlugins(pluginsWithVersions map[string][]string) bool {
	valid := true
	allPlugins := map[string][]plugins.Plugin{}

	for rootPluginName, dependentPluginNames := range pluginsWithVersions {
		if _, err := plugins.New(rootPluginName); err != nil {
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid root plugin name '%s'", rootPluginName))
			valid = false
		}

		dependentPlugins := []plugins.Plugin{}
		for _, pluginName := range dependentPluginNames {
			if p, err := plugins.New(pluginName); err != nil {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid dependent plugin name '%s' in root plugin '%s'", pluginName, rootPluginName))
				valid = false
			} else {
				dependentPlugins = append(dependentPlugins, *p)
			}
		}

		allPlugins[rootPluginName] = dependentPlugins
	}

	if valid {
		return plugins.VerifyDependencies(allPlugins)
	}

	return valid
}

func (r *ReconcileJenkinsBaseConfiguration) verifyBackup() (bool, error) {
	if r.jenkins.Spec.Backup == "" {
		r.logger.V(log.VWarn).Info("Backup strategy not set in 'spec.backup'")
		return false, nil
	}

	valid := false
	for _, backupType := range virtuslabv1alpha1.AllowedJenkinsBackups {
		if r.jenkins.Spec.Backup == backupType {
			valid = true
		}
	}

	if !valid {
		r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid backup strategy '%s'", r.jenkins.Spec.Backup))
		r.logger.V(log.VWarn).Info(fmt.Sprintf("Allowed backups '%+v'", virtuslabv1alpha1.AllowedJenkinsBackups))
		return false, nil
	}

	if r.jenkins.Spec.Backup == virtuslabv1alpha1.JenkinsBackupTypeNoBackup {
		return true, nil
	}

	backupSecretName := resources.GetBackupCredentialsSecretName(r.jenkins)
	backupSecret := &corev1.Secret{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: r.jenkins.Namespace, Name: backupSecretName}, backupSecret)
	if err != nil && errors.IsNotFound(err) {
		r.logger.V(log.VWarn).Info(fmt.Sprintf("Please create secret '%s' in namespace '%s'", backupSecretName, r.jenkins.Namespace))
		return false, nil
	} else if err != nil && !errors.IsNotFound(err) {
		return false, err
	}

	return true, nil
}
