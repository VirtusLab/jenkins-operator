package resources

import (
	"fmt"
	"text/template"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"

	"github.com/VirtusLab/jenkins-operator/pkg/controller/render"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var initBashTemplate = template.Must(template.New(initScriptName).Parse(`#!/usr/bin/env bash
set -e
set -x

# https://wiki.jenkins.io/display/JENKINS/Post-initialization+script
mkdir -p {{ .JenkinsHomePath }}/init.groovy.d
cp -n {{ .BaseConfigurationPath }}/*.groovy {{ .JenkinsHomePath }}/init.groovy.d

touch {{ .JenkinsHomePath }}/plugins.txt
cat > {{ .JenkinsHomePath }}/plugins.txt <<EOL
credentials:2.1.18
ssh-credentials:1.14
job-dsl:1.70
git:3.9.1
workflow-cps:2.61
workflow-job:2.30
workflow-aggregator:2.6
EOL

/usr/local/bin/install-plugins.sh < {{ .JenkinsHomePath }}/plugins.txt

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

	output, err := render.Render(initBashTemplate, data)
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
