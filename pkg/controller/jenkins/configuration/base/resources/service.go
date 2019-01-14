package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func buildServiceTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}
}

// NewService builds the Kubernetes service resource
func NewService(meta metav1.ObjectMeta, minikube bool) *corev1.Service {
	service := &corev1.Service{
		TypeMeta:   buildServiceTypeMeta(),
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Selector: meta.Labels,
			// The first port have to be Jenkins http port because when run with minikube
			// command 'minikube service' returns endpoints in the same sequence
			Ports: []corev1.ServicePort{
				{
					Name:       httpPortName,
					Port:       httpPortInt32,
					TargetPort: intstr.FromInt(HTTPPortInt),
				},
				{
					Name:       slavePortName,
					Port:       slavePortInt32,
					TargetPort: intstr.FromInt(slavePortInt),
				},
			},
		},
	}

	if minikube {
		// When running locally with minikube cluster Jenkins Service have to be exposed via node port
		// to allow communication operator -> Jenkins API
		service.Spec.Type = corev1.ServiceTypeNodePort
	} else {
		service.Spec.Type = corev1.ServiceTypeClusterIP
	}

	return service
}
