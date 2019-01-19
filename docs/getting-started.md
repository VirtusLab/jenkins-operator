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
  - id: jenkins-operator
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: https://github.com/VirtusLab/jenkins-operator.git
```

Watch Jenkins instance being created:

```bash
kubectl get pods -w
```

Get Jenkins credentials:

```bash
kubectl get secret jenkins-operator-credentials-example -o 'jsonpath={.data.user}' | base64 -d
kubectl get secret jenkins-operator-credentials-example -o 'jsonpath={.data.password}' | base64 -d
```

Connect to Jenkins (minikube):

```bash
minikube service jenkins-operator-example --url
```
Pick up the first URL.

Connect to Jenkins (actual Kubernetes cluster):

```bash
kubectl describe svc jenkins-operator-example
kubectl port-forward jenkins-operator-example 8080:8080
```
Then open browser with address http://localhost:8080.
![jenkins](../assets/jenkins.png)

## Configure Seed Jobs and Pipelines

Jenkins operator uses [job-dsl][job-dsl] and [ssh-credentials][ssh-credentials] plugins for configuring jobs
and deploy keys.

## Prepare job definitions and pipelines

First you have to prepare pipelines and job definition in your GitHub repository using the following structure:

```
cicd/
├── jobs
│   └── build.jenkins
└── pipelines
    └── build.jenkins
```

**cicd/jobs/build.jenkins** it's a job definition:

```
#!/usr/bin/env groovy

pipelineJob('build-jenkins-operator') {
    displayName('Build jenkins-operator')

    definition {
        cpsScm {
            scm {
                git {
                    remote {
                        url('https://github.com/VirtusLab/jenkins-operator.git')
                        credentials('jenkins-operator')
                    }
                    branches('*/master')
                }
            }
            scriptPath('cicd/pipelines/build.jenkins')
        }
    }
}
```

**cicd/jobs/build.jenkins** it's an actual Jenkins pipeline:

```
#!/usr/bin/env groovy

def label = "build-jenkins-operator-${UUID.randomUUID().toString()}"
def home = "/home/jenkins"
def workspace = "${home}/workspace/build-jenkins-operator"
def workdir = "${workspace}/src/github.com/VirtusLab/jenkins-operator/"

podTemplate(label: label,
        containers: [
                containerTemplate(name: 'jnlp', image: 'jenkins/jnlp-slave:alpine'),
                containerTemplate(name: 'go', image: 'golang:1-alpine', command: 'cat', ttyEnabled: true),
        ]) {

    node(label) {
        dir(workdir) {
            stage('Init') {
                timeout(time: 3, unit: 'MINUTES') {
                    checkout scm
                }
                container('go') {
                    sh 'apk --no-cache --update add make git gcc libc-dev'
                }
            }

            stage('Build') {
                container('go') {
                    sh 'make build'
                }
            }
        }
    }
}
```

## Configure Seed Jobs

Jenkins Seed Jobs are configured using `Jenkins.spec.seedJobs` section from your custom resource manifest:

```
apiVersion: virtuslab.com/v1alpha1
kind: Jenkins
metadata:
  name: example
spec:
  master:
   image: jenkins/jenkins:lts
  seedJobs:
  - id: jenkins-operator
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: https://github.com/VirtusLab/jenkins-operator.git
```

If your GitHub repository is **private** you have to configure corresponding **privateKey** and Kubernetes Secret:

```
apiVersion: virtuslab.com/v1alpha1
kind: Jenkins
metadata:
  name: example
spec:
  master:
   image: jenkins/jenkins:lts
  seedJobs:
  - id: jenkins-operator
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: git@github.com:VirtusLab/jenkins-operator.git
    privateKey:
      secretKeyRef:
        name: deploy-keys
        key: jenkins-operator
```

And Kubernetes Secret:

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

**jenkins-operator** will automatically discover and configure all seed jobs.

You can verify if deploy keys were successfully configured in Jenkins **Credentials** tab.

![jenkins](../assets/jenkins-credentials.png)

You can verify if your pipelines were successfully configured in Jenkins Seed Job console output.

![jenkins](../assets/jenkins-seed.png)

## Jenkins Customisation

Jenkins can be customized using groovy scripts or configuration as code plugin. All custom configuration is stored in
the **jenkins-operator-user-configuration-example** ConfigMap which is automatically created by **jenkins-operator**. 

```
kubectl get configmap jenkins-operator-user-configuration-example -o yaml

apiVersion: v1
data:
  1-configure-theme.groovy: |2

    import jenkins.*
    import jenkins.model.*
    import hudson.*
    import hudson.model.*
    import org.jenkinsci.plugins.simpletheme.ThemeElement
    import org.jenkinsci.plugins.simpletheme.CssTextThemeElement
    import org.jenkinsci.plugins.simpletheme.CssUrlThemeElement

    Jenkins jenkins = Jenkins.getInstance()

    def decorator = Jenkins.instance.getDescriptorByType(org.codefirst.SimpleThemeDecorator.class)

    List<ThemeElement> configElements = new ArrayList<>();
    configElements.add(new CssTextThemeElement("DEFAULT"));
    configElements.add(new CssUrlThemeElement("https://cdn.rawgit.com/afonsof/jenkins-material-theme/gh-pages/dist/material-light-green.css"));
    decorator.setElements(configElements);
    decorator.save();

    jenkins.save()
kind: ConfigMap
metadata:
  labels:
    app: jenkins-operator
    jenkins-cr: example
    watch: "true"
  name: jenkins-operator-user-configuration-example
  namespace: default
``` 

When **jenkins-operator-user-configuration-example** ConfigMap is updated Jenkins automatically runs the **jenkins-operator-user-configuration** Jenkins Job which executes all scripts.

## Install Plugins

To install a plugin please add **2-install-slack-plugin.groovy** script to the **jenkins-operator-user-configuration-example** ConfigMap:

```
apiVersion: v1
data:
  1-configure-theme.groovy: |2

    import jenkins.*
    import jenkins.model.*
    import hudson.*
    import hudson.model.*
    import org.jenkinsci.plugins.simpletheme.ThemeElement
    import org.jenkinsci.plugins.simpletheme.CssTextThemeElement
    import org.jenkinsci.plugins.simpletheme.CssUrlThemeElement

    Jenkins jenkins = Jenkins.getInstance()

    def decorator = Jenkins.instance.getDescriptorByType(org.codefirst.SimpleThemeDecorator.class)

    List<ThemeElement> configElements = new ArrayList<>();
    configElements.add(new CssTextThemeElement("DEFAULT"));
    configElements.add(new CssUrlThemeElement("https://cdn.rawgit.com/afonsof/jenkins-material-theme/gh-pages/dist/material-light-green.css"));
    decorator.setElements(configElements);
    decorator.save();

    jenkins.save()
  2-install-slack-plugin.groovy: |2
  
    import jenkins.model.*
    import java.util.logging.Level
    import java.util.logging.Logger

    def instance = Jenkins.getInstance()
    def plugins = instance.getPluginManager()
    def updateCenter = instance.getUpdateCenter()
    def hasInstalledPlugins = false

    Logger logger = Logger.getLogger('jenkins.instance.restart')

    if (!plugins.getPlugin("slack")) {
        logger.log(Level.INFO, "Installing plugin: slack")

        updateCenter.updateAllSites()
        def plugin = updateCenter.getPlugin("slack")
        def installResult = plugin.deploy()
        while (!installResult.isDone()) sleep(10)
        hasInstalledPlugins = true 
        instance.save()
    }

    if (hasInstalledPlugins) {
        logger.log(Level.INFO, "Successfully installed slack plugin, restarting ...")
        // Queue a restart of the instance
        instance.save()
        instance.doSafeRestart(null)
    } else {
        logger.log(Level.INFO, "No plugins need installing.")
    }
```

Then **jenkins-operator** will automatically trigger **jenkins-operator-user-configuration** Jenkins Job again.

## Configure Backup & Restore (work in progress)

Not implemented yet.

## Debugging

Turn on debug in **jenkins-operator** deployment:

```bash
sed -i 's|\(args:\).*|\1\ ["--debug"\]|' deploy/operator.yaml
kubectl apply -f deploy/operator.yaml
```

Watch Kubernetes events:

```bash
kubectl get events --sort-by='{.lastTimestamp}'
```

Verify Jenkins master logs:

```bash
kubectl logs -f jenkins-master-example
```

Verify jenkins-operator logs:

```bash
kubectl logs deployment/jenkins-operator
```

## Troubleshooting

Delete Jenkins master pod and wait for the new one to come up:

```bash
kubectl delete pod jenkins-operator-example
```

[job-dsl]:https://github.com/jenkinsci/job-dsl-plugin
[ssh-credentials]:https://github.com/jenkinsci/ssh-credentials-plugin