package groovy

import (
	"crypto/sha256"
	"encoding/base64"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/jobs"

	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ExecuteGroovyJobName this is the fixed execute groovy job name
	ExecuteGroovyJobName = "Execute Groovy Scripts"

	groovyScriptParameterName = "GROOVY_SCRIPT"
)

// Groovy defines API for groovy scripts execution via jenkins job
type Groovy struct {
	jenkinsClient jenkinsclient.Jenkins
	k8sClient     k8s.Client
	logger        logr.Logger
}

// New creates new instance of Groovy
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger) *Groovy {
	return &Groovy{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
	}
}

// ConfigureGroovyJob configures jenkins job for executing groovy scripts
func (g *Groovy) ConfigureGroovyJob() error {
	_, err := g.jenkinsClient.CreateOrUpdateJob(groovyJobConfigXML, ExecuteGroovyJobName)
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

	parameters := map[string]string{
		groovyScriptParameterName: groovyScript,
	}

	done, err := jobsClient.EnsureBuildJob(ExecuteGroovyJobName, encodedHash, parameters, jenkins, true)
	if err != nil {
		return false, err
	}
	return done, nil
}

// FIXME(antoniaklja) use mask-password plugin for params.GROOVY_SCRIPT
// TODO add groovy script name
var groovyJobConfigXML = `
<flow-definition plugin="workflow-job@2.30">
  <actions/>
  <description></description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <hudson.model.ParametersDefinitionProperty>
      <parameterDefinitions>
        <hudson.model.TextParameterDefinition>
          <name>` + groovyScriptParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.TextParameterDefinition>
      </parameterDefinitions>
    </hudson.model.ParametersDefinitionProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61">
    <script>import hudson.util.RemotingDiagnostics
import jenkins.model.Jenkins.MasterComputer

println RemotingDiagnostics.executeGroovy(&quot;&quot;&quot;
${params.GROOVY_SCRIPT}
&quot;&quot;&quot;, MasterComputer.localChannel)</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
`
