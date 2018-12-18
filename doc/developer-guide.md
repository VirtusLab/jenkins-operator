# Developer guide

This document explains how to setup your dev environment.

## Prerequisites

- [dep][dep_tool] version v0.5.0+
- [git][git_tool]
- [go][go_tool] version v1.10+
- [minikube][minikube] version v0.31.0+ (preferred Hypervisor - [virtualbox][virtualbox])
- [docker][docker_tool] version 17.03+

## Clone repository and download dependencies

```bash
mkdir -p $GOPATH/src/github.com/VirtusLab
cd $GOPATH/src/github.com/VirtusLab/
git clone git@github.com:VirtusLab/jenkins-operator.git
cd jenkins-operator
make go-dependecies
```

## Build and run

Build and run `jenkins-operator` locally:

```bash
make build && make docker-build && make minikube-run EXTRA_ARGS='--minikube --local'
```

Once minikube and `jenkins-operator` are up and running, apply CR file:

```bash
kubectl apply -f jenkins-operator/deploy/crds/virtuslab_v1alpha1_jenkins_cr.yaml
kubectl get jenkins -o yaml
kubectl get po
```

## Testing

Run unit tests:

```bash
make test
```

Run e2e tests with minikube:

```bash
make minikube-run
eval $(minikube docker-env)
make docker-build
make e2e E2E_IMAGE=jenkins-operator
```

See the project [README][jenkins-operator] for more details.

## Hacks

### `pkg/apis/virtuslab/v1alpha1/jenkins_types` has changed

Generate deepcopy using operator-sdk:

```bash
operator-sdk generate k8s 
```

output should be simillar to:

```
INFO[0000] Running code-generation for Custom Resource group versions: [virtuslab:v1alpha1, ] 
Generating deepcopy funcs
INFO[0001] Code-generation complete.
```

### Getting Jenkins URL and basic credentials

```bash
minikube service jenkins-operator-example --url
kubectl get secret jenkins-operator-credentials-example -o yaml
```

### Install custom plugins

Extend `initBashTemplate` in `jenkins-operator/pkg/controller/jenkins/configuration/base/resources/scripts_configmap.go`:

```
touch {{ .JenkinsHomePath }}/plugins.txt
cat > {{ .JenkinsHomePath }}/plugins.txt <<EOL
credentials:2.1.18
ssh-credentials:1.14
job-dsl:1.70
git:3.9.1
mask-passwords:2.12.0
workflow-cps:2.61
workflow-job:2.30
workflow-aggregator:2.6
EOL
```


[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[repo_sdk]:https://github.com/operator-framework/operator-sdk
[fork_guide]:https://help.github.com/articles/fork-a-repo/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube]:https://kubernetes.io/docs/tasks/tools/install-minikube/
[virtualbox]:https://www.virtualbox.org/wiki/Downloads
[jenkins-operator]:../README.md