package backup

import (
	"context"
	"fmt"
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/backup/aws"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/backup/nobackup"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/jobs"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugins"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	restoreJobName = constants.OperatorName + "-restore-backup"
)

// Provider defines API of backup providers
type Provider interface {
	GetRestoreJobXML(jenkins virtuslabv1alpha1.Jenkins) (string, error)
	GetBackupJobXML(jenkins virtuslabv1alpha1.Jenkins) (string, error)
	IsConfigurationValidForBasePhase(jenkins virtuslabv1alpha1.Jenkins, logger logr.Logger) bool
	IsConfigurationValidForUserPhase(k8sClient k8s.Client, jenkins virtuslabv1alpha1.Jenkins, logger logr.Logger) (bool, error)
	GetRequiredPlugins() map[string][]plugins.Plugin
}

// Backup defines backup manager which is responsible of backup of jobs history
type Backup struct {
	jenkins       *virtuslabv1alpha1.Jenkins
	k8sClient     k8s.Client
	logger        logr.Logger
	jenkinsClient jenkinsclient.Jenkins
}

// New returns instance of backup manager
func New(jenkins *virtuslabv1alpha1.Jenkins, k8sClient k8s.Client, logger logr.Logger, jenkinsClient jenkinsclient.Jenkins) *Backup {
	return &Backup{jenkins: jenkins, k8sClient: k8sClient, logger: logger, jenkinsClient: jenkinsClient}
}

// EnsureRestoreJob creates and updates Jenkins job used to restore backup
func (b *Backup) EnsureRestoreJob() error {
	if b.jenkins.Status.UserConfigurationCompletedTime == nil {
		provider, err := GetBackupProvider(b.jenkins.Spec.Backup)
		if err != nil {
			return err
		}
		restoreJobXML, err := provider.GetRestoreJobXML(*b.jenkins)
		if err != nil {
			return err
		}
		_, created, err := b.jenkinsClient.CreateOrUpdateJob(restoreJobXML, restoreJobName)
		if err != nil {
			return err
		}
		if created {
			b.logger.Info(fmt.Sprintf("'%s' job has been created", restoreJobName))
		}

		return nil
	}

	return nil
}

// RestoreBackup restores backup
func (b *Backup) RestoreBackup() (reconcile.Result, error) {
	if !b.jenkins.Status.BackupRestored && b.jenkins.Status.UserConfigurationCompletedTime == nil {
		jobsClient := jobs.New(b.jenkinsClient, b.k8sClient, b.logger)

		hash := "hash-restore" // it can be hardcoded because restore job can be run only once
		done, err := jobsClient.EnsureBuildJob(restoreJobName, hash, map[string]string{}, b.jenkins, true)
		if err != nil {
			// build failed and can be recovered - retry build and requeue reconciliation loop with timeout
			if err == jobs.ErrorBuildFailed {
				return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
			}
			// build failed and cannot be recovered
			if err == jobs.ErrorUnrecoverableBuildFailed {
				b.logger.Info(fmt.Sprintf("Restore backup can not be performed. Please check backup configuration in CR and credentials in secret '%s'.", resources.GetBackupCredentialsSecretName(b.jenkins)))
				b.logger.Info(fmt.Sprintf("You can also check '%s' job logs in Jenkins", constants.BackupJobName))
				return reconcile.Result{}, nil
			}
			// unexpected error - requeue reconciliation loop
			return reconcile.Result{}, err
		}
		// build not finished yet - requeue reconciliation loop with timeout
		if !done {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
		}

		b.jenkins.Status.BackupRestored = true
		err = b.k8sClient.Update(context.TODO(), b.jenkins)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// EnsureBackupJob creates and updates Jenkins job used to backup
func (b *Backup) EnsureBackupJob() error {
	provider, err := GetBackupProvider(b.jenkins.Spec.Backup)
	if err != nil {
		return err
	}
	backupJobXML, err := provider.GetBackupJobXML(*b.jenkins)
	if err != nil {
		return err
	}
	_, created, err := b.jenkinsClient.CreateOrUpdateJob(backupJobXML, constants.BackupJobName)
	if err != nil {
		return err
	}
	if created {
		b.logger.Info(fmt.Sprintf("'%s' job has been created", constants.BackupJobName))
	}

	return nil
}

// GetBackupProvider returns backup provider by type
func GetBackupProvider(backupType virtuslabv1alpha1.JenkinsBackup) (Provider, error) {
	switch backupType {
	case virtuslabv1alpha1.JenkinsBackupTypeNoBackup:
		return &nobackup.NoBackup{}, nil
	case virtuslabv1alpha1.JenkinsBackupTypeAmazonS3:
		return &aws.AmazonS3Backup{}, nil
	default:
		return nil, errors.Errorf("Invalid BackupManager type '%s'", backupType)
	}
}

// GetPluginsRequiredByAllBackupProviders returns plugins required by all backup providers
func GetPluginsRequiredByAllBackupProviders() map[string][]plugins.Plugin {
	allPlugins := map[string][]plugins.Plugin{}
	for _, provider := range getAllProviders() {
		for key, value := range provider.GetRequiredPlugins() {
			allPlugins[key] = value
		}
	}

	return allPlugins
}

func getAllProviders() []Provider {
	return []Provider{
		&nobackup.NoBackup{}, &aws.AmazonS3Backup{},
	}
}
