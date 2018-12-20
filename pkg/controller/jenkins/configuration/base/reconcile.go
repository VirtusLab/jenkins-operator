package base

import (
	"context"
	"fmt"
	"reflect"
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/VirtusLab/jenkins-operator/pkg/log"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileJenkinsBaseConfiguration defines values required for Jenkins base configuration
type ReconcileJenkinsBaseConfiguration struct {
	client          client.Client
	scheme          *runtime.Scheme
	logger          logr.Logger
	jenkins         *virtuslabv1alpha1.Jenkins
	local, minikube bool
}

// New create structure which takes care of base configuration
func New(client client.Client, scheme *runtime.Scheme, logger logr.Logger,
	jenkins *virtuslabv1alpha1.Jenkins, local, minikube bool) *ReconcileJenkinsBaseConfiguration {
	return &ReconcileJenkinsBaseConfiguration{
		client:   client,
		scheme:   scheme,
		logger:   logger,
		jenkins:  jenkins,
		local:    local,
		minikube: minikube,
	}
}

// Reconcile takes care of base configuration
func (r *ReconcileJenkinsBaseConfiguration) Reconcile() (*reconcile.Result, jenkinsclient.Jenkins, error) {
	if !r.validate(r.jenkins) {
		r.logger.V(log.VWarn).Info("Please correct Jenkins CR")
		return &reconcile.Result{}, nil, nil
	}

	metaObject := resources.NewResourceObjectMeta(r.jenkins)

	if err := r.createOperatorCredentialsSecret(metaObject); err != nil {
		return &reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Operator credentials secret is present")

	if err := r.createScriptsConfigMap(metaObject); err != nil {
		return &reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Scripts config map is present")

	if err := r.createBaseConfigurationConfigMap(metaObject); err != nil {
		return &reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Base configuration config map is present")

	if err := r.createService(metaObject); err != nil {
		return &reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Service is present")

	result, err := r.createJenkinsMasterPod(metaObject)
	if err != nil {
		return &reconcile.Result{}, nil, err
	}
	if result != nil {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is present")

	result, err = r.waitForJenkins(metaObject)
	if err != nil {
		return &reconcile.Result{}, nil, err
	}
	if result != nil {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is ready")

	jenkinsClient, err := r.getJenkinsClient(metaObject)
	if err != nil {
		return &reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Jenkins API client set")

	return nil, jenkinsClient, nil
}

func (r *ReconcileJenkinsBaseConfiguration) createOperatorCredentialsSecret(meta metav1.ObjectMeta) error {
	found := &corev1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.jenkins), Namespace: r.jenkins.ObjectMeta.Namespace}, found)

	if err != nil && apierrors.IsNotFound(err) {
		return r.createResource(resources.NewOperatorCredentialsSecret(meta, r.jenkins))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if found.Data[resources.OperatorCredentialsSecretUserNameKey] != nil &&
		found.Data[resources.OperatorCredentialsSecretPasswordKey] != nil {
		return nil
	}

	return r.updateResource(resources.NewOperatorCredentialsSecret(meta, r.jenkins))
}

func (r *ReconcileJenkinsBaseConfiguration) createScriptsConfigMap(meta metav1.ObjectMeta) error {
	scripts, err := resources.NewScriptsConfigMap(meta, r.jenkins)
	if err != nil {
		return err
	}
	return r.createOrUpdateResource(scripts)
}

func (r *ReconcileJenkinsBaseConfiguration) createBaseConfigurationConfigMap(meta metav1.ObjectMeta) error {
	scripts, err := resources.NewBaseConfigurationConfigMap(meta, r.jenkins)
	if err != nil {
		return err
	}
	return r.createOrUpdateResource(scripts)
}

func (r *ReconcileJenkinsBaseConfiguration) createService(meta metav1.ObjectMeta) error {
	err := r.createResource(resources.NewService(&meta, r.minikube))
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) getJenkinsMasterPod(meta metav1.ObjectMeta) (*corev1.Pod, error) {
	jenkinsMasterPod := resources.NewJenkinsMasterPod(meta, r.jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPod.Name, Namespace: jenkinsMasterPod.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err
	}
	return currentJenkinsMasterPod, nil
}

func (r *ReconcileJenkinsBaseConfiguration) createJenkinsMasterPod(meta metav1.ObjectMeta) (*reconcile.Result, error) {
	// Check if this Pod already exists
	currentJenkinsMasterPod, err := r.getJenkinsMasterPod(meta)
	if err != nil && errors.IsNotFound(err) {
		jenkinsMasterPod := resources.NewJenkinsMasterPod(meta, r.jenkins)
		r.logger.Info(fmt.Sprintf("Creating a new Jenkins Master Pod %s/%s", jenkinsMasterPod.Namespace, jenkinsMasterPod.Name))
		err = r.createResource(jenkinsMasterPod)
		if err != nil {
			return nil, err
		}
		if r.jenkins.Status.BaseConfigurationCompletedTime != nil {
			r.jenkins.Status.BaseConfigurationCompletedTime = nil
			err = r.updateResource(r.jenkins)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	} else if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	// Recreate pod
	recreatePod := false
	if currentJenkinsMasterPod != nil &&
		(currentJenkinsMasterPod.Status.Phase == corev1.PodFailed ||
			currentJenkinsMasterPod.Status.Phase == corev1.PodSucceeded ||
			currentJenkinsMasterPod.Status.Phase == corev1.PodUnknown) {
		r.logger.Info(fmt.Sprintf("Invalid Jenkins pod phase %v, recreating pod", currentJenkinsMasterPod.Status.Phase))
		recreatePod = true
	}

	if currentJenkinsMasterPod != nil &&
		r.jenkins.Spec.Master.Image != currentJenkinsMasterPod.Spec.Containers[0].Image {
		r.logger.Info(fmt.Sprintf("Jenkins image has changed to %v, recreating pod", r.jenkins.Spec.Master.Image))
		recreatePod = true
	}

	if currentJenkinsMasterPod != nil && len(r.jenkins.Spec.Master.Annotations) > 0 &&
		!reflect.DeepEqual(r.jenkins.Spec.Master.Annotations, currentJenkinsMasterPod.ObjectMeta.Annotations) {
		r.logger.Info(fmt.Sprintf("Jenkins pod annotations have changed to %v, recreating pod", r.jenkins.Spec.Master.Annotations))
		recreatePod = true
	}

	if currentJenkinsMasterPod != nil &&
		!reflect.DeepEqual(r.jenkins.Spec.Master.Resources, currentJenkinsMasterPod.Spec.Containers[0].Resources) {
		r.logger.Info(fmt.Sprintf("Jenkins pod resources have changed to %v, recreating pod", r.jenkins.Spec.Master.Resources))
		recreatePod = true
	}

	if currentJenkinsMasterPod != nil && recreatePod && currentJenkinsMasterPod.ObjectMeta.DeletionTimestamp == nil {
		r.logger.Info(fmt.Sprintf("Terminating Jenkins Master Pod %s/%s", currentJenkinsMasterPod.Namespace, currentJenkinsMasterPod.Name))
		if err := r.client.Delete(context.TODO(), currentJenkinsMasterPod); err != nil {
			return nil, err
		}
		return &reconcile.Result{Requeue: true}, nil
	}

	return nil, nil
}

func (r *ReconcileJenkinsBaseConfiguration) waitForJenkins(meta metav1.ObjectMeta) (*reconcile.Result, error) {
	jenkinsMasterPodStatus, err := r.getJenkinsMasterPod(meta)
	if err != nil {
		return nil, err
	}

	if jenkinsMasterPodStatus.ObjectMeta.DeletionTimestamp != nil {
		r.logger.Info("Jenkins master pod is terminating")
		return &reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	if jenkinsMasterPodStatus.Status.Phase != corev1.PodRunning {
		r.logger.Info("Jenkins master pod not ready")
		return &reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	for _, containerStatus := range jenkinsMasterPodStatus.Status.ContainerStatuses {
		if !containerStatus.Ready {
			r.logger.Info("Jenkins master pod not ready, readiness probe failed")
			return &reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
		}
	}

	return nil, nil
}

func (r *ReconcileJenkinsBaseConfiguration) getJenkinsClient(meta metav1.ObjectMeta) (jenkinsclient.Jenkins, error) {
	jenkinsURL, err := jenkinsclient.BuildJenkinsAPIUrl(
		r.jenkins.ObjectMeta.Namespace, meta.Name, resources.HTTPPortInt, r.local, r.minikube)
	if err != nil {
		return nil, err
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Jenkins API URL %s", jenkinsURL))

	credentialsSecret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.jenkins), Namespace: r.jenkins.ObjectMeta.Namespace}, credentialsSecret)
	if err != nil {
		return nil, err
	}
	currentJenkinsMasterPod, err := r.getJenkinsMasterPod(meta)
	if err != nil {
		return nil, err
	}

	var tokenCreationTime *time.Time
	tokenCreationTimeBytes := credentialsSecret.Data[resources.OperatorCredentialsSecretTokenCreationKey]
	if tokenCreationTimeBytes != nil {
		tokenCreationTime = &time.Time{}
		err = tokenCreationTime.UnmarshalText(tokenCreationTimeBytes)
		if err != nil {
			tokenCreationTime = nil
		}

	}
	if credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey] == nil ||
		tokenCreationTimeBytes == nil || tokenCreationTime == nil ||
		currentJenkinsMasterPod.ObjectMeta.CreationTimestamp.Time.UTC().After(tokenCreationTime.UTC()) {
		r.logger.Info("Generating Jenkins API token for operator")
		userName := string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey])
		jenkinsClient, err := jenkinsclient.New(
			jenkinsURL,
			userName,
			string(credentialsSecret.Data[resources.OperatorCredentialsSecretPasswordKey]))
		if err != nil {
			return nil, err
		}

		token, err := jenkinsClient.GenerateToken(userName, "token")
		if err != nil {
			return nil, err
		}

		credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey] = []byte(token.GetToken())
		now, _ := time.Now().UTC().MarshalText()
		credentialsSecret.Data[resources.OperatorCredentialsSecretTokenCreationKey] = now
		err = r.updateResource(credentialsSecret)
		if err != nil {
			return nil, err
		}
	}

	return jenkinsclient.New(
		jenkinsURL,
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey]),
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey]))
}
