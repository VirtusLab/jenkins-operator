package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsSpec defines the desired state of Jenkins
type JenkinsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Backup         JenkinsBackup         `json:"backup,omitempty"`
	BackupAmazonS3 JenkinsBackupAmazonS3 `json:"backupAmazonS3,omitempty"`
	Master         JenkinsMaster         `json:"master,omitempty"`
	SeedJobs       []SeedJob             `json:"seedJobs,omitempty"`
}

// JenkinsBackup defines type of Jenkins backup
type JenkinsBackup string

const (
	// JenkinsBackupTypeNoBackup tells that Jenkins won't backup jobs
	JenkinsBackupTypeNoBackup = "NoBackup"
	// JenkinsBackupTypeAmazonS3 tells that Jenkins will backup jobs into AWS S3 bucket
	JenkinsBackupTypeAmazonS3 = "AmazonS3"
)

// AllowedJenkinsBackups consists allowed Jenkins backup types
var AllowedJenkinsBackups = []JenkinsBackup{JenkinsBackupTypeNoBackup, JenkinsBackupTypeAmazonS3}

// JenkinsBackupAmazonS3 defines backup configuration to AWS S3 bucket
type JenkinsBackupAmazonS3 struct {
	BucketName string `json:"bucketName,omitempty"`
	BucketPath string `json:"bucketPath,omitempty"`
	Region     string `json:"region,omitempty"`
}

// JenkinsMaster defines the Jenkins master pod attributes and plugins,
// every single change requires Jenkins master pod restart
type JenkinsMaster struct {
	Image       string                      `json:"image,omitempty"`
	Annotations map[string]string           `json:"masterAnnotations,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Plugins     map[string][]string         `json:"plugins,omitempty"`
}

// JenkinsStatus defines the observed state of Jenkins
type JenkinsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	BackupRestored                 bool         `json:"backupRestored,omitempty"`
	BaseConfigurationCompletedTime *metav1.Time `json:"baseConfigurationCompletedTime,omitempty"`
	UserConfigurationCompletedTime *metav1.Time `json:"userConfigurationCompletedTime,omitempty"`
	Builds                         []Build      `json:"builds,omitempty"`
}

// BuildStatus defines type of Jenkins build job status
type BuildStatus string

const (
	// BuildSuccessStatus - the build had no errors
	BuildSuccessStatus BuildStatus = "success"
	// BuildUnstableStatus - the build had some errors but they were not fatal. For example, some tests failed
	BuildUnstableStatus BuildStatus = "unstable"
	// BuildNotBuildStatus - this status code is used in a multi-stage build (like maven2) where a problem in earlier stage prevented later stages from building
	BuildNotBuildStatus BuildStatus = "not_build"
	// BuildFailureStatus - the build had a fatal error
	BuildFailureStatus BuildStatus = "failure"
	// BuildAbortedStatus - the build was manually aborted
	BuildAbortedStatus BuildStatus = "aborted"
	// BuildRunningStatus - this is custom build status for running build, not present in jenkins build result
	BuildRunningStatus BuildStatus = "running"
	// BuildExpiredStatus - this is custom build status for expired build, not present in jenkins build result
	BuildExpiredStatus BuildStatus = "expired"
)

// Build defines Jenkins Build status with corresponding metadata
type Build struct {
	JobName        string       `json:"jobName,omitempty"`
	Hash           string       `json:"hash,omitempty"`
	Number         int64        `json:"number,omitempty"`
	Status         BuildStatus  `json:"status,omitempty"`
	Retires        int          `json:"retries,omitempty"`
	CreateTime     *metav1.Time `json:"createTime,omitempty"`
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Jenkins is the Schema for the jenkins API
// +k8s:openapi-gen=true
type Jenkins struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsSpec   `json:"spec,omitempty"`
	Status JenkinsStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsList contains a list of Jenkins
type JenkinsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jenkins `json:"items"`
}

// SeedJob defined configuration for seed jobs and deploy keys
type SeedJob struct {
	ID               string     `json:"id"`
	Description      string     `json:"description,omitempty"`
	Targets          string     `json:"targets,omitempty"`
	RepositoryBranch string     `json:"repositoryBranch,omitempty"`
	RepositoryURL    string     `json:"repositoryUrl"`
	PrivateKey       PrivateKey `json:"privateKey,omitempty"`
}

// PrivateKey contains a private key
type PrivateKey struct {
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef"`
}

func init() {
	SchemeBuilder.Register(&Jenkins{}, &JenkinsList{})
}
