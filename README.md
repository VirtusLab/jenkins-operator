# Jenkins Operator

## What's Jenkins Operator?

Jenkins operator it's a Kubernetes native operator which fully manages Jenkins on Kubernetes.
It was built with immutability and declarative configuration as code in mind.

It provides out of the box:
- integration with Kubernetes
- pipelines as code
- extensibility via groovy scripts or configuration as code plugin
- security and hardening

## Problem statement and goals

The main reason why we decided to write the **jenkins-operator** is the fact that we faced a lot of problems with standard Jenkins deployment.
We want to make Jenkins more robust, suitable for dynamic and multi-tenant environments. 

Some of the problems we want to solve:
- volumes handling (AWS EBS volume attach/detach issue when using PVC)
- installing plugins with incompatible versions or security vulnerabilities
- better configuration as code
- lack of end to end tests
- handle graceful shutdown properly
- security and hardening out of the box
- orphaned jobs with no jnlp connection
- make errors more visible for end users
- backup and restore for jobs history

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
* create Jenkins Jobs View for all jobs managed by the operator
* code clean up and more tests

Base configuration:
* TLS/SSL configuration

User configuration:
* backup and restore for Jenkins jobs running as standalone job (AWS, GCP, Azure)
* trigger backup job before pod deletion using preStop k8s hooks
* verify Jenkins configuration events

[installation]:doc/installation.md
[getting_started]:doc/getting-started.md
[how_it_works]:doc/how-it-works.md
[developer_guide]:doc/developer-guide.md
