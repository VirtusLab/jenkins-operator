package resources

import (
	"fmt"
	"text/template"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var initBashTemplate = template.Must(template.New(initScriptName).Parse(`#!/usr/bin/env bash
set -e
set -x

# https://wiki.jenkins.io/display/JENKINS/Post-initialization+script
mkdir -p {{ .JenkinsHomePath }}/init.groovy.d
cp -n {{ .BaseConfigurationPath }}/*.groovy {{ .JenkinsHomePath }}/init.groovy.d

/sbin/tini -s -- /usr/local/bin/jenkins.sh
`))

func buildConfigMapTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	}
}

func buildInitBashScript() (*string, error) {
	data := struct {
		JenkinsHomePath       string
		BaseConfigurationPath string
	}{
		JenkinsHomePath:       jenkinsHomePath,
		BaseConfigurationPath: jenkinsBaseConfigurationVolumePath,
	}

	output, err := renderTemplate(initBashTemplate, data)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

func getScriptsConfigMapName(jenkins *virtuslabv1alpha1.Jenkins) string {
	return fmt.Sprintf("jenkins-operator-scripts-%s", jenkins.ObjectMeta.Name)
}

// NewScriptsConfigMap builds Kubernetes config map used to store scripts
func NewScriptsConfigMap(meta metav1.ObjectMeta, jenkins *virtuslabv1alpha1.Jenkins) (*corev1.ConfigMap, error) {
	meta.Name = getScriptsConfigMapName(jenkins)

	initBashScript, err := buildInitBashScript()
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		TypeMeta:   buildConfigMapTypeMeta(),
		ObjectMeta: meta,
		Data: map[string]string{
			initScriptName: *initBashScript,
		},
	}, nil
}
