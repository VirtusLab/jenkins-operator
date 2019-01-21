package event

import (
	"fmt"

	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

const (
	// Information only and will not cause any problems
	TypeNormal = Type("Normal")
	// These events are to warn that something might go wrong
	TypeWarning = Type("Warning")
)

type Type string
type Reason string

type Recorder interface {
	Emit(object runtime.Object, eventType Type, reason Reason, message string)
	Emitf(object runtime.Object, eventType Type, reason Reason, format string, args ...interface{})
}

type recorder struct {
	recorder record.EventRecorder
}

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
	eventBroadcaster.StartLogging(glog.Infof)
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
