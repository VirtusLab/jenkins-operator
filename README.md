# jenkins-operator

Kubernetes native Jenkins operator which manages Jenkins on Kubernetes.
It was built with immutability and declarative configuration as code in mind.

It provides out of the box:
- preconfigured Kubernetes plugin
- Seed Jobs and DSL Pipelines as code
- integration with Configuration as Code plugin
- secure and hardened Jenkins instance
- basic authentication with username and password

The main reason why we decided to write the Jenkins operator is the fact that we faced a lot of problems with standard Jenkins deployment.
Also, we heavily use Jenkins on large production Kubernetes clusters in a multi-tenant environment.

Some of the problems we want to solve:
- volumes handling (AWS EBS volume attach/detach issue)
- installing plugins with incompatible versions or security vulnerabilities
- declarative configuration as code
- lack of end to end tests
- make Jenkins more suitable for dynamic environment and handle graceful shutdown properly
- security and hardening out of the box
- job stuck and jnlp connection is lost
- make errors more visible for end users

## Documentation

1. [Installation][installation]
2. [Getting Started][getting_started]
3. [How it works][how_it_works]
4. [Developer Guide][developer_guide]

## Contribution

Feel free to file [issues](https://github.com/VirtusLab/jenkins-operator/issues) or [pull requests](https://github.com/VirtusLab/jenkins-operator/pulls).    

## TODO

Common:
* simple API for generating Kubernetes events using one common format
* ~~VirtusLab docker registry~~ https://hub.docker.com/r/virtuslab/jenkins-operator
* ~~decorate Jenkins API client and add more functions for handling jobs and builds e.g. Ensure, CreateOrUpdate~~
* documentation
* ~~VirtusLab flavored Jenkins theme~~
* create Jenkins Jobs View for all jobs managed by the operator
* ~~jenkins job for executing groovy scripts~~

Base configuration:
* ~~install configuration as a code Jenkins plugin~~
* handle Jenkins restart when base configuration has changed
* ~~install~~ and configure Kubernetes plugin (in-progress)
* e2e pipelines using Kubernetes plugin
* Jenkins hardening, disable insecure options

User configuration:
* ~~user reconciliation loop with CR validation~~
* ~~configure seed jobs and deploy keys~~
* ~~e2e tests for seed jobs~~
* configure Jenkins authorization (via configuration as a code plugin or groovy scripts)
* backup and restore for Jenkins jobs running as standalone job (AWS, GCP, Azure)
* trigger backup job before pod deletion using preStop k8s hooks
* verify Jenkins configuration events

[installation]:doc/installation.md
[getting_started]:doc/getting-started.md
[how_it_works]:doc/how-it-works.md
[developer_guide]:doc/developer-guide.md
