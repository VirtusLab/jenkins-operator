# jenkins-operator

Kubernetes native Jenkins operator.

## Developer guide

Can be found [here][developer_guide].

## TODO

Common:
- simple library for sending Kubernetes events
- implement Jenkins.Status in custom resource

Base configuration:
- install configuration as a code Jenkins plugin
- restart Jenkins when scripts config map or base configuration config map have changed
- install and configure Kubernetes plugin
- disable insecure options

User configuration:
- user reconciliation loop (work in progress)
- configure seed jobs and deploy keys (work in progress)
- e2e tests for seed jobs
- backup and restore for Jenkins jobs running as standalone job
- trigger backup job before pod deletion using preStop k8s hooks
- verify Jenkins configuration events

[developer_guide]:doc/developer-guide.md