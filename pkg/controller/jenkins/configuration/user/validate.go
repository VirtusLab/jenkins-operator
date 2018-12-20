package user

import (
	"context"
	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func (r *ReconcileUserConfiguration) validate(k8sClient k8s.Client, jenkins *virtuslabv1alpha1.Jenkins) bool {
	// validate jenkins.Spec.SeedJobs
	if jenkins.Spec.SeedJobs != nil {
		for _, seedJob := range jenkins.Spec.SeedJobs {

			// validate seed job id is not empty
			if len(seedJob.ID) == 0 {
				r.logger.V(0).Info("seed job id can't be empty")
				return false
			}

			// validate repository url match private key
			if strings.Contains(seedJob.RepositoryURL, "git@") {
				if seedJob.PrivateKey.SecretKeyRef == nil {
					r.logger.V(0).Info("private key can't be empty while using ssh repository url")
					return false
				}
			}

			// validate private key from secret
			if seedJob.PrivateKey.SecretKeyRef != nil {
				deployKeySecret := &v1.Secret{}
				namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: seedJob.PrivateKey.SecretKeyRef.Name}
				err := k8sClient.Get(context.TODO(), namespaceName, deployKeySecret)
				if err != nil {
					r.logger.V(0).Info("couldn't read private key secret")
					return false
				}

				privateKey := string(deployKeySecret.Data[seedJob.PrivateKey.SecretKeyRef.Key])
				if privateKey != "" {
					r.logger.V(0).Info("private key is empty")
					return false
				}

				if !strings.HasPrefix(privateKey, "-----BEGIN RSA PRIVATE KEY-----") {
					r.logger.V(0).Info("private key has wrong prefix")
					return false
				}
			}
		}
	}
	return true
}
