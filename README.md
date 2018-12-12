# jenkins-operator

Kubernetes native Jenkins operator

## Developer guide

Can be found [here][developer_guide]

## TODO

- send Kubernetes events

Base configuration:
- install configuration as a code Jenkins plugin
- restart Jenkins when scripts config map or base configuration config map have changed
- install and configure Kubernetes plugin
- disable insecure options

User configuration:
- AWS s3 restore backup job
- AWS s3 backup job
- create and run seed jobs
- apply custom configuration by configuration as a code Jenkins plugin
- trigger backup job before pod deletion
- verify Jenkins configuration events

[developer_guide]:doc/developer-guide.md