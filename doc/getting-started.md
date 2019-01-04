# Getting Started

1. [First Steps]()
2. [Deploy Jenkins]()
3. [Configure Seed Jobs and Pipelines]()
4. [Install Plugins]()
5. [Configure Authorization]()
6. [Configure Backup & Restore]()
7. [Debugging]()

## First Steps

// TODO install operator etc.

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

Create Jenkins Custom Resource:

```bash
kubectl create -f deploy/crds/virtuslab_v1alpha1_jenkins_cr.yaml
```

Watch Jenkins instance being created:

```bash
kubectl get pods -w
```

Connect to Jenkins:

// TODO (antoniaklja)

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

[job-dsl]:https://github.com/jenkinsci/job-dsl-plugin
[ssh-credentials]:https://github.com/jenkinsci/ssh-credentials-plugin

## Install Plugins

## Configure Authorization

## Configure Backup & Restore

## Debugging