package user

import (
	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkins "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/user/seedjobs"
	"github.com/VirtusLab/jenkins-operator/pkg/log"
	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileUserConfiguration defines values required for Jenkins user configuration
type ReconcileUserConfiguration struct {
	k8sClient     k8s.Client
	jenkinsClient jenkins.Jenkins
	logger        logr.Logger
	jenkins       *virtuslabv1alpha1.Jenkins
}

// New create structure which takes care of user configuration
func New(k8sClient k8s.Client, jenkinsClient jenkins.Jenkins, logger logr.Logger,
	jenkins *virtuslabv1alpha1.Jenkins) *ReconcileUserConfiguration {
	return &ReconcileUserConfiguration{
		k8sClient:     k8sClient,
		jenkinsClient: jenkinsClient,
		logger:        logger,
		jenkins:       jenkins,
	}
}

// Reconcile it's a main reconciliation loop for user supplied configuration
func (r *ReconcileUserConfiguration) Reconcile() (*reconcile.Result, error) {
	if !r.validate(r.k8sClient, r.jenkins) {
		r.logger.V(log.VWarn).Info("Please correct Jenkins CR")
		return &reconcile.Result{}, nil
	}

	err := seedjobs.EnsureSeedJobs(r.jenkinsClient, r.k8sClient, r.jenkins)
	if err != nil {
		return &reconcile.Result{}, err
	}

	return nil, nil
}
