package base

import (
	"regexp"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"

	docker "github.com/docker/distribution/reference"
)

var (
	dockerImageRegexp = regexp.MustCompile(`^` + docker.TagRegexp.String() + `$`)
)

func (r *ReconcileJenkinsBaseConfiguration) validate(jenkins *virtuslabv1alpha1.Jenkins) bool {
	if jenkins.Spec.Master.Image == "" {
		r.logger.V(0).Info("Image not set")
		return false
	}

	if !dockerImageRegexp.MatchString(jenkins.Spec.Master.Image) && !docker.ReferenceRegexp.MatchString(jenkins.Spec.Master.Image) {
		r.logger.V(0).Info("Invalid image")
		return false
	}

	return true
}
