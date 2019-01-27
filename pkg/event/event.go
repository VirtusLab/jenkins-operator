package event

import (
	"fmt"

	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

const (
	// TypeNormal is the information event type
	TypeNormal = Type("Normal")
	// TypeWarning is the warning event type, informs that something went wrong
	TypeWarning = Type("Warning")
)

// Type is the type of event
type Type string

// Reason is the type of reason message, used in evant
type Reason string

// Recorder is the interface used to emit events
type Recorder interface {
	Emit(object runtime.Object, eventType Type, reason Reason, message string)
	Emitf(object runtime.Object, eventType Type, reason Reason, format string, args ...interface{})
}

type recorder struct {
	recorder record.EventRecorder
}

// New returns recorder used to emit events
func New(config *rest.Config) (Recorder, error) {
	eventRecorder, err := initializeEventRecorder(config)
	if err != nil {
		return nil, err
	}

	return &recorder{
		recorder: eventRecorder,
	}, nil
}

func initializeEventRecorder(config *rest.Config) (record.EventRecorder, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	eventBroadcaster := record.NewBroadcaster()
	//eventBroadcaster.StartLogging(glog.Infof) TODO integrate with proper logger
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: client.CoreV1().Events("")})
	eventRecorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		v1.EventSource{
			Component: constants.OperatorName})
	return eventRecorder, nil
}

func (r recorder) Emit(object runtime.Object, eventType Type, reason Reason, message string) {
	r.recorder.Event(object, string(eventType), string(reason), message)
}

func (r recorder) Emitf(object runtime.Object, eventType Type, reason Reason, format string, args ...interface{}) {
	r.recorder.Event(object, string(eventType), string(reason), fmt.Sprintf(format, args...))
}
