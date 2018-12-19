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
	Master   JenkinsMaster `json:"master,omitempty"`
	SeedJobs []SeedJob     `json:"seedJobs,omitempty"`
}

// JenkinsMaster defines the Jenkins master pod attributes
type JenkinsMaster struct {
	Image       string                      `json:"image,omitempty"`
	Annotations map[string]string           `json:"masterAnnotations,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
}

// JenkinsStatus defines the observed state of Jenkins
type JenkinsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	BaseConfigurationCompletedTime *metav1.Time `json:"baseConfigurationCompletedTime,omitempty"`
	UserConfigurationCompletedTime *metav1.Time `json:"userConfigurationCompletedTime,omitempty"`
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
