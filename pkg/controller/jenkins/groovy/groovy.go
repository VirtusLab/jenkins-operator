package groovy

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/jobs"

	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	jobHashParameterName = "hash"
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
	_, created, err := g.jenkinsClient.CreateOrUpdateJob(fmt.Sprintf(configurationJobXMLFmt, g.scriptsPath), g.jobName)
	if err != nil {
		return err
	}
	if created {
		g.logger.Info(fmt.Sprintf("'%s' job has been created", g.jobName))
	}
	return nil
}

// EnsureGroovyJob executes groovy script and verifies jenkins job status according to reconciliation loop lifecycle
func (g *Groovy) EnsureGroovyJob(secretOrConfigMapData map[string]string, jenkins *virtuslabv1alpha1.Jenkins) (bool, error) {
	jobsClient := jobs.New(g.jenkinsClient, g.k8sClient, g.logger)

	hash := g.calculateHash(secretOrConfigMapData)
	done, err := jobsClient.EnsureBuildJob(g.jobName, hash, map[string]string{jobHashParameterName: hash}, jenkins, true)
	if err != nil {
		return false, err
	}
	return done, nil
}

func (g *Groovy) calculateHash(secretOrConfigMapData map[string]string) string {
	hash := sha256.New()

	var keys []string
	for key := range secretOrConfigMapData {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		hash.Write([]byte(key))
		hash.Write([]byte(secretOrConfigMapData[key]))
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

const configurationJobXMLFmt = `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.31">
  <actions/>
  <description></description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <org.jenkinsci.plugins.workflow.job.properties.DisableConcurrentBuildsJobProperty/>
    <hudson.model.ParametersDefinitionProperty>
      <parameterDefinitions>
        <hudson.model.StringParameterDefinition>
          <name>` + jobHashParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
      </parameterDefinitions>
    </hudson.model.ParametersDefinitionProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61">
    <script>def scriptsPath = &apos;%s&apos;
def expectedHash = params.hash

node(&apos;master&apos;) {
    def scriptsText = sh(script: &quot;ls ${scriptsPath} | sort&quot;, returnStdout: true).trim()
    def scripts = []
    scripts.addAll(scriptsText.tokenize(&apos;\n&apos;))
    
    stage(&apos;Synchronizing files&apos;) {
        def complete = false
        for(int i = 1; i &lt;= 10; i++) {
            def actualHash = calculateHash((String[])scripts, scriptsPath)
            println &quot;Expected hash &apos;${expectedHash}&apos;, actual hash &apos;${actualHash}&apos;&quot;
            if(expectedHash == actualHash) {
                complete = true
                break
            }
            sleep 2
        }
        if(!complete) {
            error(&quot;Timeout while synchronizing files&quot;)
        }
    }
    
    for(script in scripts) {
        stage(script) {
            load &quot;${scriptsPath}/${script}&quot;
        }
    }
}

@NonCPS
def calculateHash(String[] scripts, String scriptsPath) {
    def hash = java.security.MessageDigest.getInstance(&quot;SHA-256&quot;)
    for(script in scripts) {
        hash.update(script.getBytes())
        def fileLocation = java.nio.file.Paths.get(&quot;${scriptsPath}/${script}&quot;)
        def fileData = java.nio.file.Files.readAllBytes(fileLocation)
        hash.update(fileData)
    }
    return Base64.getEncoder().encodeToString(hash.digest())
}</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
`
