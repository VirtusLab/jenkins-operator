package user

import (
	"context"
	"fmt"
	"strings"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/log"

	"crypto/x509"
	"encoding/pem"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Validate validates Jenkins CR Spec section
func (r *ReconcileUserConfiguration) Validate(jenkins *virtuslabv1alpha1.Jenkins) bool {
	// validate jenkins.Spec.SeedJobs
	if jenkins.Spec.SeedJobs != nil {
		for _, seedJob := range jenkins.Spec.SeedJobs {
			logger := r.logger.WithValues("seedJob", fmt.Sprintf("%+v", seedJob)).V(log.VWarn)

			// validate seed job id is not empty
			if len(seedJob.ID) == 0 {
				logger.Info("seed job id can't be empty")
				return false
			}

			// validate repository url match private key
			if strings.Contains(seedJob.RepositoryURL, "git@") {
				if seedJob.PrivateKey.SecretKeyRef == nil {
					logger.Info("private key can't be empty while using ssh repository url")
					return false
				}
			}

			// validate private key from secret
			if seedJob.PrivateKey.SecretKeyRef != nil {
				deployKeySecret := &v1.Secret{}
				namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: seedJob.PrivateKey.SecretKeyRef.Name}
				err := r.k8sClient.Get(context.TODO(), namespaceName, deployKeySecret)
				//TODO(bantoniak) handle error properly
				if err != nil {
					logger.Info("couldn't read private key secret")
					return false
				}

				privateKey := string(deployKeySecret.Data[seedJob.PrivateKey.SecretKeyRef.Key])
				if privateKey == "" {
					logger.Info("private key is empty")
					return false
				}

				if !validatePrivateKey(privateKey) {
					logger.Info("private key is invalid")
					return false
				}
			}
		}
	}
	return true
}

func validatePrivateKey(privateKey string) bool {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return false
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return false
	}

	err = priv.Validate()
	if err != nil {
		return false
	}

	return true
}
