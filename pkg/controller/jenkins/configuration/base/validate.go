package base

import (
	"fmt"
	"regexp"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugin"
	"github.com/VirtusLab/jenkins-operator/pkg/log"

	docker "github.com/docker/distribution/reference"
)

var (
	dockerImageRegexp = regexp.MustCompile(`^` + docker.TagRegexp.String() + `$`)
)

// Validate validates Jenkins CR Spec.master section
func (r *ReconcileJenkinsBaseConfiguration) Validate(jenkins *virtuslabv1alpha1.Jenkins) bool {
	if jenkins.Spec.Master.Image == "" {
		r.logger.V(log.VWarn).Info("Image not set")
		return false
	}

	if !dockerImageRegexp.MatchString(jenkins.Spec.Master.Image) && !docker.ReferenceRegexp.MatchString(jenkins.Spec.Master.Image) {
		r.logger.V(log.VWarn).Info("Invalid image")
		return false

	}

	if !r.validatePlugins(jenkins.Spec.Master.Plugins) {
		return false
	}

	return true
}

func (r *ReconcileJenkinsBaseConfiguration) validatePlugins(plugins map[string][]string) bool {
	valid := true
	allPlugins := map[string][]plugin.Plugin{}

	for rootPluginName, dependentPluginNames := range plugins {
		if _, err := plugin.New(rootPluginName); err != nil {
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid root plugin name '%s'", rootPluginName))
			valid = false
		}

		dependentPlugins := []plugin.Plugin{}
		for _, pluginName := range dependentPluginNames {
			if p, err := plugin.New(pluginName); err != nil {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid dependent plugin name '%s' in root plugin '%s'", pluginName, rootPluginName))
				valid = false
			} else {
				dependentPlugins = append(dependentPlugins, *p)
			}
		}

		allPlugins[rootPluginName] = dependentPlugins
	}

	if valid {
		return plugin.VerifyDependencies(allPlugins)
	}

	return valid
}
