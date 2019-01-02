package user

import (
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/user/seedjobs"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/jobs"

	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileUserConfiguration defines values required for Jenkins user configuration
type ReconcileUserConfiguration struct {
	k8sClient     k8s.Client
	jenkinsClient jenkinsclient.Jenkins
	logger        logr.Logger
	jenkins       *virtuslabv1alpha1.Jenkins
}

// New create structure which takes care of user configuration
func New(k8sClient k8s.Client, jenkinsClient jenkinsclient.Jenkins, logger logr.Logger,
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
	// reconcile seed jobs
	result, err := r.reconcileSeedJobs()
	if err != nil {
		return result, err
	}
	return result, nil

	return nil, nil
}

func (r *ReconcileUserConfiguration) reconcileSeedJobs() (*reconcile.Result, error) {
	seedJobs := seedjobs.New(r.jenkinsClient, r.k8sClient, r.logger)
	done, err := seedJobs.EnsureSeedJobs(r.jenkins)
	if err != nil {
		// build failed and can be recovered - retry build and requeue reconciliation loop with timeout
		if err == jobs.ErrorBuildFailed {
			return &reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
		}
		// build failed and cannot be recovered
		if err == jobs.ErrorUncoverBuildFailed {
			return nil, nil
		}
		// unexpected error - requeue reconciliation loop
		return &reconcile.Result{}, err
	}
	// build not finished yet - requeue reconciliation loop with timeout
	if !done {
		return &reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
	}
	return nil, nil
}
