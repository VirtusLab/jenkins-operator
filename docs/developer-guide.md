# Developer guide

This document explains how to setup your development environment.

## Prerequisites

- [operator_sdk][operator_sdk]
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
make go-dependencies
```

## Build and run

Build and run **jenkins-operator** locally:

```bash
make build && make minikube-run EXTRA_ARGS='--minikube --local'
```

Once minikube and **jenkins-operator** are up and running, apply Jenkins custom resource:

```bash
kubectl apply -f deploy/crds/virtuslab_v1alpha1_jenkins_cr.yaml
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
make start-minikube
eval $(minikube docker-env)
make e2e
```

## Tips & Tricks

### Building docker image on minikube (for e2e tests)

To be able to work with the docker daemon on `minikube` machine run the following command before building an image:

```bash
eval $(minikube docker-env)
```

### When `pkg/apis/virtuslab/v1alpha1/jenkins_types.go` has changed

Run:

```bash
make deepcopy-gen
```

### Getting Jenkins URL and basic credentials

```bash
minikube service jenkins-operator-example --url
kubectl get secret jenkins-operator-credentials-example -o 'jsonpath={.data.user}' | base64 -d
kubectl get secret jenkins-operator-credentials-example -o 'jsonpath={.data.password}' | base64 -d
```


[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[operator_sdk]:https://github.com/operator-framework/operator-sdk
[fork_guide]:https://help.github.com/articles/fork-a-repo/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube]:https://kubernetes.io/docs/tasks/tools/install-minikube/
[virtualbox]:https://www.virtualbox.org/wiki/Downloads
[jenkins-operator]:../README.md