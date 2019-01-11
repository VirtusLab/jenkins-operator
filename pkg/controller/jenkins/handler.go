package jenkins

import (
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// enqueueRequestForJenkins enqueues a Request for secrets and configmaps created by jenkins-operator.
type enqueueRequestForJenkins struct{}

func (e *enqueueRequestForJenkins) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if req := e.getOwnerReconcileRequests(evt.Meta); req != nil {
		q.Add(*req)
	}
}

func (e *enqueueRequestForJenkins) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if req := e.getOwnerReconcileRequests(evt.MetaOld); req != nil {
		q.Add(*req)
	}
	if req := e.getOwnerReconcileRequests(evt.MetaNew); req != nil {
		q.Add(*req)
	}
}

func (e *enqueueRequestForJenkins) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if req := e.getOwnerReconcileRequests(evt.Meta); req != nil {
		q.Add(*req)
	}
}

func (e *enqueueRequestForJenkins) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if req := e.getOwnerReconcileRequests(evt.Meta); req != nil {
		q.Add(*req)
	}
}

func (e *enqueueRequestForJenkins) getOwnerReconcileRequests(object metav1.Object) *reconcile.Request {
	if object.GetLabels()[constants.LabelAppKey] == constants.LabelAppValue &&
		object.GetLabels()[constants.LabelWatchKey] == constants.LabelWatchValue &&
		len(object.GetLabels()[constants.LabelJenkinsCRKey]) > 0 {
		return &reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetLabels()[constants.LabelJenkinsCRKey],
		}}
	}

	return nil
}
