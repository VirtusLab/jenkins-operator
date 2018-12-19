package user

import (
	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"strings"
)

func (r *ReconcileUserConfiguration) validate(jenkins *virtuslabv1alpha1.Jenkins) bool {
	// validate jenkins.Spec.SeedJobs
	if jenkins.Spec.SeedJobs != nil {
		for _, seedJob := range jenkins.Spec.SeedJobs {
			if len(seedJob.ID) == 0 {
				r.logger.V(0).Info("seed job id can't be empty")
				return false
			}

			if strings.Contains(seedJob.RepositoryURL, "git@") {
				if seedJob.PrivateKey.SecretKeyRef == nil {
					r.logger.V(0).Info("private key can't be empty while using ssh repository url")
					return false
				}
			}
		}
	}
	return true
}
