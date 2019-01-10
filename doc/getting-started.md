# Getting Started

This document describes a getting started guide for **jenkins-operator** and an additional configuration.

1. [First Steps](#first-steps)
2. [Deploy Jenkins](#deploy-jenkins)
3. [Configure Seed Jobs and Pipelines](#configure-seed-jobs-and-pipelines)
4. [Install Plugins](#install-plugins)
5. [Configure Authorization](#configure-authorization)
6. [Configure Backup & Restore](#configure-backup-&-restore)
7. [Debugging](#debugging)

## First Steps

Prepare your Kubernetes cluster and set up access.
Once you have running Kubernetes cluster you can focus on installing **jenkins-operator** according to the [Installation](installation.md) guide.

## Deploy Jenkins

Once jenkins-operator is up and running let's deploy actual Jenkins instance.
Let's use example below:

```bash
apiVersion: virtuslab.com/v1alpha1
kind: Jenkins
metadata:
  name: example
spec:
  master:
   image: jenkins/jenkins
  seedJobs:
  - id: jenkins-operator-e2e
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator e2e tests repository"
    repositoryBranch: master
    repositoryUrl: https://github.com/VirtusLab/jenkins-operator-e2e.git
```

Watch Jenkins instance being created:

```bash
kubectl get pods -w
```

Get Jenkins credentials:

```bash
kubectl get secret jenkins-operator-credentials-example -o 'jsonpath={.data.password}' | base64 -d
```

Connect to Jenkins (minikube):

```bash
minikube service jenkins-operator-example --url
```

Connect to Jenkins (actual Kubernetes cluster):

```bash
kubectl describe svc jenkins-operator-example
kubectl jenkins-operator-example 8080:8080

```

![jenkins](../jenkins.png)

## Configure Seed Jobs and Pipelines

Jenkins operator uses [job-dsl][job-dsl] and [ssh-credentials][ssh-credentials] plugins for configuring seed jobs
and deploy keys.


It can be configured using `Jenkins.spec.seedJobs` section from custom resource manifest:

```
apiVersion: virtuslab.com/v1alpha1
kind: Jenkins
metadata:
  name: example
spec:
  master:
   image: jenkins/jenkins
  seedJobs:
  - id: jenkins-operator
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator e2e tests repository"
    repositoryBranch: master
    repositoryUrl: git@github.com:VirtusLab/jenkins-operator-e2e.git
    privateKey:
      secretKeyRef:
        name: deploy-keys
        key: jenkins-operator-e2e
```

And corresponding Kubernetes Secret (in the same namespace) with private key:

```
apiVersion: v1
kind: Secret
metadata:
  name: deploy-keys
data:
  jenkins-operator-e2e: |
    -----BEGIN RSA PRIVATE KEY-----
    MIIJKAIBAAKCAgEAxxDpleJjMCN5nusfW/AtBAZhx8UVVlhhhIKXvQ+dFODQIdzO
    oDXybs1zVHWOj31zqbbJnsfsVZ9Uf3p9k6xpJ3WFY9b85WasqTDN1xmSd6swD4N8
    ...
```

If your GitHub repository is public, you don't have to configure `privateKey` and create Kubernetes Secret:

```
apiVersion: virtuslab.com/v1alpha1
kind: Jenkins
metadata:
  name: example
spec:
  master:
   image: jenkins/jenkins
  seedJobs:
  - id: jenkins-operator-e2e
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator e2e tests repository"
    repositoryBranch: master
    repositoryUrl: https://github.com/VirtusLab/jenkins-operator-e2e.git
```

Jenkins operator will automatically configure and trigger Seed Job Pipeline for all entries from `Jenkins.spec.seedJobs`.

## Install Plugins

## Configure Authorization

## Configure Backup & Restore

## Debugging

[job-dsl]:https://github.com/jenkinsci/job-dsl-plugin
[ssh-credentials]:https://github.com/jenkinsci/ssh-credentials-plugin