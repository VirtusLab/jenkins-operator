package seedjobs

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkinsclient "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/jobs"
	"github.com/VirtusLab/jenkins-operator/pkg/log"

	"github.com/go-logr/logr"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ConfigureSeedJobsName this is the fixed seed job name
	ConfigureSeedJobsName = constants.OperatorName + "-configure-seed-job"

	deployKeyIDParameterName      = "DEPLOY_KEY_ID"
	privateKeyParameterName       = "PRIVATE_KEY"
	repositoryURLParameterName    = "REPOSITORY_URL"
	repositoryBranchParameterName = "REPOSITORY_BRANCH"
	targetsParameterName          = "TARGETS"
	displayNameParameterName      = "SEED_JOB_DISPLAY_NAME"
)

// SeedJobs defines API for configuring and ensuring Jenkins Seed Jobs and Deploy Keys
type SeedJobs struct {
	jenkinsClient jenkinsclient.Jenkins
	k8sClient     k8s.Client
	logger        logr.Logger
}

// New creates SeedJobs object
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger) *SeedJobs {
	return &SeedJobs{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
	}
}

// EnsureSeedJobs configures seed job and runs it for every entry from Jenkins.Spec.SeedJobs
func (s *SeedJobs) EnsureSeedJobs(jenkins *virtuslabv1alpha1.Jenkins) (done bool, err error) {
	err = s.createJob()
	if err != nil {
		s.logger.V(log.VWarn).Info("Couldn't create jenkins seed job")
		return false, err
	}
	done, err = s.buildJobs(jenkins)
	if err != nil {
		s.logger.V(log.VWarn).Info("Couldn't build jenkins seed job")
		return false, err
	}
	return done, nil
}

// createJob is responsible for creating jenkins job which configures jenkins seed jobs and deploy keys
func (s *SeedJobs) createJob() error {
	_, created, err := s.jenkinsClient.CreateOrUpdateJob(seedJobConfigXML, ConfigureSeedJobsName)
	if err != nil {
		return err
	}
	if created {
		s.logger.Info(fmt.Sprintf("'%s' job has been created", ConfigureSeedJobsName))
	}
	return nil
}

// buildJobs is responsible for running jenkins builds which configures jenkins seed jobs and deploy keys
func (s *SeedJobs) buildJobs(jenkins *virtuslabv1alpha1.Jenkins) (done bool, err error) {
	allDone := true
	seedJobs := jenkins.Spec.SeedJobs
	for _, seedJob := range seedJobs {
		privateKey, err := s.privateKeyFromSecret(jenkins.Namespace, seedJob)
		if err != nil {
			return false, err
		}
		parameters := map[string]string{
			deployKeyIDParameterName:      seedJob.ID,
			privateKeyParameterName:       privateKey,
			repositoryURLParameterName:    seedJob.RepositoryURL,
			repositoryBranchParameterName: seedJob.RepositoryBranch,
			targetsParameterName:          seedJob.Targets,
			displayNameParameterName:      fmt.Sprintf("Seed Job from %s", seedJob.ID),
		}

		hash := sha256.New()
		hash.Write([]byte(parameters[deployKeyIDParameterName]))
		hash.Write([]byte(parameters[privateKeyParameterName]))
		hash.Write([]byte(parameters[repositoryURLParameterName]))
		hash.Write([]byte(parameters[repositoryBranchParameterName]))
		hash.Write([]byte(parameters[targetsParameterName]))
		hash.Write([]byte(parameters[displayNameParameterName]))
		encodedHash := base64.URLEncoding.EncodeToString(hash.Sum(nil))

		jobsClient := jobs.New(s.jenkinsClient, s.k8sClient, s.logger)
		done, err := jobsClient.EnsureBuildJob(ConfigureSeedJobsName, encodedHash, parameters, jenkins, true)
		if err != nil {
			return false, err
		}
		if !done {
			allDone = false
		}
	}
	return allDone, nil
}

// privateKeyFromSecret it's utility function which extracts deploy key from the kubernetes secret
func (s *SeedJobs) privateKeyFromSecret(namespace string, seedJob virtuslabv1alpha1.SeedJob) (string, error) {
	if seedJob.PrivateKey.SecretKeyRef != nil {
		deployKeySecret := &v1.Secret{}
		namespaceName := types.NamespacedName{Namespace: namespace, Name: seedJob.PrivateKey.SecretKeyRef.Name}
		err := s.k8sClient.Get(context.TODO(), namespaceName, deployKeySecret)
		if err != nil {
			return "", err
		}
		return string(deployKeySecret.Data[seedJob.PrivateKey.SecretKeyRef.Key]), nil
	}
	return "", nil
}

// FIXME(antoniaklja) use mask-password plugin for params.PRIVATE_KEY
// seedJobConfigXML this is the XML representation of seed job
var seedJobConfigXML = `
<flow-definition plugin="workflow-job@2.30">
  <actions/>
  <description>Configure Seed Jobs</description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <hudson.model.ParametersDefinitionProperty>
      <parameterDefinitions>
        <hudson.model.StringParameterDefinition>
          <name>` + deployKeyIDParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + privateKeyParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + repositoryURLParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + repositoryBranchParameterName + `</name>
          <description></description>
          <defaultValue>master</defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + displayNameParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + targetsParameterName + `</name>
          <description></description>
          <defaultValue>cicd/jobs/*.jenkins</defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
      </parameterDefinitions>
    </hudson.model.ParametersDefinitionProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61">
    <script>import com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey
import com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey.DirectEntryPrivateKeySource
import com.cloudbees.plugins.credentials.CredentialsScope
import com.cloudbees.plugins.credentials.SystemCredentialsProvider
import com.cloudbees.plugins.credentials.domains.Domain
import hudson.model.FreeStyleProject
import hudson.model.labels.LabelAtom
import hudson.plugins.git.BranchSpec
import hudson.plugins.git.GitSCM
import hudson.plugins.git.SubmoduleConfig
import hudson.plugins.git.extensions.impl.CloneOption
import javaposse.jobdsl.plugin.ExecuteDslScripts
import javaposse.jobdsl.plugin.LookupStrategy
import javaposse.jobdsl.plugin.RemovedJobAction
import javaposse.jobdsl.plugin.RemovedViewAction
import jenkins.model.Jenkins
import javaposse.jobdsl.plugin.GlobalJobDslSecurityConfiguration
import jenkins.model.GlobalConfiguration

import static com.google.common.collect.Lists.newArrayList

// https://javadoc.jenkins.io/plugin/ssh-credentials/com/cloudbees/jenkins/plugins/sshcredentials/impl/BasicSSHUserPrivateKey.html
BasicSSHUserPrivateKey deployKeyPrivate = new BasicSSHUserPrivateKey(
        CredentialsScope.GLOBAL,
        &quot;${params.DEPLOY_KEY_ID}&quot;,
        &quot;git&quot;,
        new DirectEntryPrivateKeySource(&quot;${params.PRIVATE_KEY}&quot;),
        &quot;&quot;,
        &quot;${params.DEPLOY_KEY_ID}&quot;
)

// https://javadoc.jenkins.io/plugin/credentials/index.html?com/cloudbees/plugins/credentials/SystemCredentialsProvider.html
SystemCredentialsProvider.getInstance().getStore().addCredentials(Domain.global(), deployKeyPrivate)

Jenkins jenkins = Jenkins.instance

def jobDslSeedName = &quot;${params.DEPLOY_KEY_ID}-` + constants.SeedJobSuffix + `&quot;
def jobDslDeployKeyName = &quot;${params.DEPLOY_KEY_ID}&quot;
def jobRef = jenkins.getItem(jobDslSeedName)

def repoList = GitSCM.createRepoList(&quot;${params.REPOSITORY_URL}&quot;, jobDslDeployKeyName)
def gitExtensions = [new CloneOption(true, true, &quot;&quot;, 10)]
def scm = new GitSCM(
        repoList,
        newArrayList(new BranchSpec(&quot;${params.REPOSITORY_BRANCH}&quot;)),
        false,
        Collections.&lt;SubmoduleConfig&gt; emptyList(),
        null,
        null,
        gitExtensions
)

def executeDslScripts = new ExecuteDslScripts()
executeDslScripts.setTargets(&quot;${params.TARGETS}&quot;)
executeDslScripts.setSandbox(false)
executeDslScripts.setRemovedJobAction(RemovedJobAction.DELETE)
executeDslScripts.setRemovedViewAction(RemovedViewAction.DELETE)
executeDslScripts.setLookupStrategy(LookupStrategy.SEED_JOB)
executeDslScripts.setAdditionalClasspath(&quot;src&quot;)

if (jobRef == null) {
        jobRef = jenkins.createProject(FreeStyleProject, jobDslSeedName)
}
jobRef.getBuildersList().clear()
jobRef.getBuildersList().add(executeDslScripts)
jobRef.setDisplayName(&quot;${params.SEED_JOB_DISPLAY_NAME}&quot;)
jobRef.setScm(scm)
jobRef.setAssignedLabel(new LabelAtom(&quot;master&quot;))

// disable Job DSL script approval
GlobalConfiguration.all().get(GlobalJobDslSecurityConfiguration.class).useScriptSecurity=false
GlobalConfiguration.all().get(GlobalJobDslSecurityConfiguration.class).save()
jenkins.getQueue().schedule(jobRef)
</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
`
