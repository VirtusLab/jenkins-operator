# Developer guide

This document explains how to setup your dev environment.

## Prerequisites
- [dep][dep_tool] version v0.5.0+
- [git][git_tool]
- [go][go_tool] version v1.10+

## Download Operator SDK

Go to the [jenkins-operator repo][jenkins-operator] and follow the [fork guide][fork_guide] to fork, clone, and setup the local jenkins-operator repository.

## Vendor dependencies

Run the following in the project root directory to update the vendored dependencies:

```sh
$ cd $GOPATH/src/github.com/VirtusLab/jenkins-operator
$ make go-dependencies
```

## Build the Operator

Build the Operator `jenkins-operator` binary:

```sh
$ make build
```

Build the Operator `jenkins-operator` docker image:

```sh
$ make build
$ make docker-build
```

## Run

Run locally with minikube:

```sh
$ make minikube-run EXTRA_ARGS='--minikube --local'
```

## Testing

Run unit tests:

```sh
$ make test
```

Run e2e tests with minikube:

```sh
$ make minikube-run
$ eval $(minikube docker-env)
$ make docker-build
$ make e2e E2E_IMAGE=<docker-image-builded-locally>
```

**Note:** running all tests requires:
- [docker][docker_tool] version 17.03+
- [kubectl][kubectl_tool] version v1.10.0+
- [minikube][minikube] version v0.30.0+(preferred Hypervisor - [virtualbox][virtualbox])

See the project [README][jenkins-operator] for more details.

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