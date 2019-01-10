package groovy

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/jobs"

	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// Groovy defines API for groovy scripts execution via jenkins job
type Groovy struct {
	jenkinsClient jenkinsclient.Jenkins
	k8sClient     k8s.Client
	logger        logr.Logger
	jobName       string
	scriptsPath   string
}

// New creates new instance of Groovy
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger, jobName, scriptsPath string) *Groovy {
	return &Groovy{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
		jobName:       jobName,
		scriptsPath:   scriptsPath,
	}
}

// ConfigureGroovyJob configures jenkins job for executing groovy scripts
func (g *Groovy) ConfigureGroovyJob() error {
	_, err := g.jenkinsClient.CreateOrUpdateJob(fmt.Sprintf(configurationJobXMLFmt, g.scriptsPath, g.scriptsPath), g.jobName)
	if err != nil {
		return err
	}
	return nil
}

// EnsureGroovyJob executes groovy script and verifies jenkins job status according to reconciliation loop lifecycle
// see https://wiki.jenkins.io/display/JENKINS/Jenkins+Script+Console
func (g *Groovy) EnsureGroovyJob(groovyScript string, jenkins *virtuslabv1alpha1.Jenkins) (bool, error) {
	jobsClient := jobs.New(g.jenkinsClient, g.k8sClient, g.logger)

	hash := sha256.New()
	hash.Write([]byte(groovyScript))
	encodedHash := base64.URLEncoding.EncodeToString(hash.Sum(nil))

	done, err := jobsClient.EnsureBuildJob(g.jobName, encodedHash, map[string]string{}, jenkins, true)
	if err != nil {
		return false, err
	}
	return done, nil
}

const configurationJobXMLFmt = `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.25">
  <actions/>
  <description></description>
  <keepDependencies>false</keepDependencies>
  <properties/>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.31">
    <script>import groovy.io.FileType

node(&apos;master&apos;) {
    def scriptsText = sh(script: &apos;ls %s&apos;, returnStdout: true).trim()
    def scripts = []
    scripts.addAll(scriptsText.tokenize(&apos;\n&apos;))
    for(script in scripts) {
        stage(script) {
            load &quot;%s/${script}&quot;
        }
    }
}</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
`
