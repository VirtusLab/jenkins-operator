# jenkins-operator

Kubernetes native Jenkins operator.

## Developer guide

Can be found [here][developer_guide].

## Configuration

This section describes Jenkins configuration.

### Seed Jobs

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

## TODO

Common:
- simple library for sending Kubernetes events using one common format
- decorate Jenkins API client and add more function for handling jobs e.g. Ensure, CreateOrUpdate 

Base configuration:
- install configuration as a code Jenkins plugin
- handle Jenkins restart when base configuration has changed
- install and configure Kubernetes plugin
- e2e pipelines using Kubernetes plugin
- Jenkins hardening, disable insecure options

User configuration:
- user reconciliation loop with CR validation (work in progress)
- configure seed jobs and deploy keys (work in progress)
- e2e tests for seed jobs (work in progress)
- backup and restore for Jenkins jobs running as standalone job
- trigger backup job before pod deletion using preStop k8s hooks
- verify Jenkins configuration events

[developer_guide]:doc/developer-guide.md
[job-dsl]:https://github.com/jenkinsci/job-dsl-plugin
[ssh-credentials]:https://github.com/jenkinsci/ssh-credentials-plugin