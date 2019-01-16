package base

import (
	"context"
	"fmt"
	"reflect"
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/groovy"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugin"
	"github.com/VirtusLab/jenkins-operator/pkg/log"

	"github.com/bndr/gojenkins"
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

const (
	fetchAllPlugins = 1
)

// ReconcileJenkinsBaseConfiguration defines values required for Jenkins base configuration
type ReconcileJenkinsBaseConfiguration struct {
	k8sClient       client.Client
	scheme          *runtime.Scheme
	logger          logr.Logger
	jenkins         *virtuslabv1alpha1.Jenkins
	local, minikube bool
}

// New create structure which takes care of base configuration
func New(client client.Client, scheme *runtime.Scheme, logger logr.Logger,
	jenkins *virtuslabv1alpha1.Jenkins, local, minikube bool) *ReconcileJenkinsBaseConfiguration {
	return &ReconcileJenkinsBaseConfiguration{
		k8sClient: client,
		scheme:    scheme,
		logger:    logger,
		jenkins:   jenkins,
		local:     local,
		minikube:  minikube,
	}
}

// Reconcile takes care of base configuration
func (r *ReconcileJenkinsBaseConfiguration) Reconcile() (reconcile.Result, jenkinsclient.Jenkins, error) {
	metaObject := resources.NewResourceObjectMeta(r.jenkins)

	if err := r.createOperatorCredentialsSecret(metaObject); err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Operator credentials secret is present")

	if err := r.createScriptsConfigMap(metaObject); err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Scripts config map is present")

	if err := r.createInitConfigurationConfigMap(metaObject); err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Init configuration config map is present")

	if err := r.createBaseConfigurationConfigMap(metaObject); err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Base configuration config map is present")

	if err := r.createUserConfigurationConfigMap(metaObject); err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("User configuration config map is present")

	if err := r.createRBAC(metaObject); err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("User configuration config map is present")

	if err := r.createService(metaObject); err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Service is present")

	result, err := r.createJenkinsMasterPod(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if result.Requeue {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is present")

	result, err = r.waitForJenkins(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if result.Requeue {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is ready")

	jenkinsClient, err := r.getJenkinsClient(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Jenkins API client set")

	ok, err := r.verifyBasePlugins(jenkinsClient)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if !ok {
		r.logger.V(log.VWarn).Info("Please correct Jenkins CR (spec.master.plugins)")
		currentJenkinsMasterPod, err := r.getJenkinsMasterPod(metaObject)
		if err != nil {
			return reconcile.Result{}, nil, err
		}
		if err := r.k8sClient.Delete(context.TODO(), currentJenkinsMasterPod); err != nil {
			return reconcile.Result{}, nil, err
		}
	}

	result, err = r.baseConfiguration(jenkinsClient)
	return result, jenkinsClient, err
}

func (r *ReconcileJenkinsBaseConfiguration) verifyBasePlugins(jenkinsClient jenkinsclient.Jenkins) (bool, error) {
	allPluginsInJenkins, err := jenkinsClient.GetPlugins(fetchAllPlugins)
	if err != nil {
		return false, err
	}

	var installedPlugins []string
	for _, jenkinsPlugin := range allPluginsInJenkins.Raw.Plugins {
		if !jenkinsPlugin.Deleted {
			installedPlugins = append(installedPlugins, plugin.Plugin{Name: jenkinsPlugin.ShortName, Version: jenkinsPlugin.Version}.String())
		}
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Installed plugins '%+v'", installedPlugins))

	status := true
	for rootPluginName, p := range plugin.BasePluginsMap {
		rootPlugin, _ := plugin.New(rootPluginName)
		if found, ok := isPluginInstalled(allPluginsInJenkins, *rootPlugin); !ok {
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Missing plugin '%s', actual '%+v'", rootPlugin, found))
			status = false
		}
		for _, requiredPlugin := range p {
			if found, ok := isPluginInstalled(allPluginsInJenkins, requiredPlugin); !ok {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Missing plugin '%s', actual '%+v'", requiredPlugin, found))
				status = false
			}
		}
	}

	return status, nil
}

func isPluginInstalled(plugins *gojenkins.Plugins, requiredPlugin plugin.Plugin) (gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return gojenkins.Plugin{}, false
	}

	return *p, p.Active && p.Enabled && !p.Deleted
}

func (r *ReconcileJenkinsBaseConfiguration) createOperatorCredentialsSecret(meta metav1.ObjectMeta) error {
	found := &corev1.Secret{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.jenkins), Namespace: r.jenkins.ObjectMeta.Namespace}, found)

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
	configMap, err := resources.NewScriptsConfigMap(meta, r.jenkins)
	if err != nil {
		return err
	}
	return r.createOrUpdateResource(configMap)
}

func (r *ReconcileJenkinsBaseConfiguration) createInitConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewInitConfigurationConfigMap(meta, r.jenkins)
	if err != nil {
		return err
	}
	return r.createOrUpdateResource(configMap)
}

func (r *ReconcileJenkinsBaseConfiguration) createBaseConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewBaseConfigurationConfigMap(meta, r.jenkins)
	if err != nil {
		return err
	}
	return r.createOrUpdateResource(configMap)
}

func (r *ReconcileJenkinsBaseConfiguration) createUserConfigurationConfigMap(meta metav1.ObjectMeta) error {
	currentConfigMap := &corev1.ConfigMap{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: resources.GetUserConfigurationConfigMapName(r.jenkins), Namespace: r.jenkins.Namespace}, currentConfigMap)
	if err != nil && errors.IsNotFound(err) {
		return r.k8sClient.Create(context.TODO(), resources.NewUserConfigurationConfigMap(r.jenkins))
	} else if err != nil {
		return err
	}
	//TODO make sure labels are fine

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) createRBAC(meta metav1.ObjectMeta) error {
	serviceAccount := resources.NewServiceAccount(meta)
	err := r.createResource(serviceAccount)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	role := resources.NewRole(meta)
	err = r.createOrUpdateResource(role)
	if err != nil {
		return err
	}

	roleBinding := resources.NewRoleBinding(meta)
	err = r.createOrUpdateResource(roleBinding)
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) createService(meta metav1.ObjectMeta) error {
	err := r.createResource(resources.NewService(meta, r.minikube))
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) getJenkinsMasterPod(meta metav1.ObjectMeta) (*corev1.Pod, error) {
	jenkinsMasterPod := resources.NewJenkinsMasterPod(meta, r.jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPod.Name, Namespace: jenkinsMasterPod.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err
	}
	return currentJenkinsMasterPod, nil
}

func (r *ReconcileJenkinsBaseConfiguration) createJenkinsMasterPod(meta metav1.ObjectMeta) (reconcile.Result, error) {
	// Check if this Pod already exists
	currentJenkinsMasterPod, err := r.getJenkinsMasterPod(meta)
	if err != nil && errors.IsNotFound(err) {
		jenkinsMasterPod := resources.NewJenkinsMasterPod(meta, r.jenkins)
		r.logger.Info(fmt.Sprintf("Creating a new Jenkins Master Pod %s/%s", jenkinsMasterPod.Namespace, jenkinsMasterPod.Name))
		err = r.createResource(jenkinsMasterPod)
		if err != nil {
			return reconcile.Result{}, err
		}
		r.jenkins.Status = virtuslabv1alpha1.JenkinsStatus{}
		err = r.updateResource(r.jenkins)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	} else if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	// Recreate pod
	recreatePod := false
	if currentJenkinsMasterPod != nil &&
		(currentJenkinsMasterPod.Status.Phase == corev1.PodFailed ||
			currentJenkinsMasterPod.Status.Phase == corev1.PodSucceeded ||
			currentJenkinsMasterPod.Status.Phase == corev1.PodUnknown) {
		r.logger.Info(fmt.Sprintf("Invalid Jenkins pod phase '%+v', recreating pod", currentJenkinsMasterPod.Status.Phase))
		recreatePod = true
	}

	if currentJenkinsMasterPod != nil &&
		r.jenkins.Spec.Master.Image != currentJenkinsMasterPod.Spec.Containers[0].Image {
		r.logger.Info(fmt.Sprintf("Jenkins image has changed to '%+v', recreating pod", r.jenkins.Spec.Master.Image))
		recreatePod = true
	}

	if currentJenkinsMasterPod != nil && len(r.jenkins.Spec.Master.Annotations) > 0 &&
		!reflect.DeepEqual(r.jenkins.Spec.Master.Annotations, currentJenkinsMasterPod.ObjectMeta.Annotations) {
		r.logger.Info(fmt.Sprintf("Jenkins pod annotations have changed to '%+v', recreating pod", r.jenkins.Spec.Master.Annotations))
		recreatePod = true
	}

	if currentJenkinsMasterPod != nil &&
		!reflect.DeepEqual(r.jenkins.Spec.Master.Resources, currentJenkinsMasterPod.Spec.Containers[0].Resources) {
		r.logger.Info(fmt.Sprintf("Jenkins pod resources have changed, actual '%+v' required '%+v' - recreating pod",
			currentJenkinsMasterPod.Spec.Containers[0].Resources, r.jenkins.Spec.Master.Resources))
		recreatePod = true
	}

	if currentJenkinsMasterPod != nil && recreatePod && currentJenkinsMasterPod.ObjectMeta.DeletionTimestamp == nil {
		r.logger.Info(fmt.Sprintf("Terminating Jenkins Master Pod %s/%s", currentJenkinsMasterPod.Namespace, currentJenkinsMasterPod.Name))
		if err := r.k8sClient.Delete(context.TODO(), currentJenkinsMasterPod); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsBaseConfiguration) waitForJenkins(meta metav1.ObjectMeta) (reconcile.Result, error) {
	jenkinsMasterPodStatus, err := r.getJenkinsMasterPod(meta)
	if err != nil {
		return reconcile.Result{}, err
	}

	if jenkinsMasterPodStatus.ObjectMeta.DeletionTimestamp != nil {
		r.logger.V(log.VDebug).Info("Jenkins master pod is terminating")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	if jenkinsMasterPodStatus.Status.Phase != corev1.PodRunning {
		r.logger.V(log.VDebug).Info("Jenkins master pod not ready")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	for _, containerStatus := range jenkinsMasterPodStatus.Status.ContainerStatuses {
		if !containerStatus.Ready {
			r.logger.V(log.VDebug).Info("Jenkins master pod not ready, readiness probe failed")
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsBaseConfiguration) getJenkinsClient(meta metav1.ObjectMeta) (jenkinsclient.Jenkins, error) {
	jenkinsURL, err := jenkinsclient.BuildJenkinsAPIUrl(
		r.jenkins.ObjectMeta.Namespace, meta.Name, resources.HTTPPortInt, r.local, r.minikube)
	if err != nil {
		return nil, err
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Jenkins API URL %s", jenkinsURL))

	credentialsSecret := &corev1.Secret{}
	err = r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.jenkins), Namespace: r.jenkins.ObjectMeta.Namespace}, credentialsSecret)
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

func (r *ReconcileJenkinsBaseConfiguration) baseConfiguration(jenkinsClient jenkinsclient.Jenkins) (reconcile.Result, error) {
	groovyClient := groovy.New(jenkinsClient, r.k8sClient, r.logger, fmt.Sprintf("%s-base-configuration", constants.OperatorName), resources.JenkinsBaseConfigurationVolumePath)

	err := groovyClient.ConfigureGroovyJob()
	if err != nil {
		return reconcile.Result{}, err
	}

	configuration := &corev1.ConfigMap{}
	namespaceName := types.NamespacedName{Namespace: r.jenkins.Namespace, Name: resources.GetBaseConfigurationConfigMapName(r.jenkins)}
	err = r.k8sClient.Get(context.TODO(), namespaceName, configuration)
	if err != nil {
		return reconcile.Result{}, err
	}

	done, err := groovyClient.EnsureGroovyJob(configuration.Data, r.jenkins)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !done {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
	}

	return reconcile.Result{}, nil
}
