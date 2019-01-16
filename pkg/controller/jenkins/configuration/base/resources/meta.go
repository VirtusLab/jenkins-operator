package resources

import (
	"fmt"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewResourceObjectMeta builds ObjectMeta for all Kubernetes resources created by operator
func NewResourceObjectMeta(jenkins *virtuslabv1alpha1.Jenkins) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      GetResourceName(jenkins),
		Namespace: jenkins.ObjectMeta.Namespace,
		Labels:    BuildResourceLabels(jenkins),
	}
}

// BuildResourceLabels returns labels for all Kubernetes resources created by operator
func BuildResourceLabels(jenkins *virtuslabv1alpha1.Jenkins) map[string]string {
	return map[string]string{
		constants.LabelAppKey:       constants.LabelAppValue,
		constants.LabelJenkinsCRKey: jenkins.Name,
	}
}

// BuildLabelsForWatchedResources returns labels for Kubernetes resources which operator want to watch
// resources with that labels should not be deleted after Jenkins CR deletion, to prevent this situation don't set
// any owner
func BuildLabelsForWatchedResources(jenkins *virtuslabv1alpha1.Jenkins) map[string]string {
	return map[string]string{
		constants.LabelAppKey:       constants.LabelAppValue,
		constants.LabelJenkinsCRKey: jenkins.Name,
		constants.LabelWatchKey:     constants.LabelWatchValue,
	}
}

// GetResourceName returns name of Kubernetes resource base on Jenkins CR
func GetResourceName(jenkins *virtuslabv1alpha1.Jenkins) string {
	return fmt.Sprintf("%s-%s", constants.LabelAppValue, jenkins.ObjectMeta.Name)
}
