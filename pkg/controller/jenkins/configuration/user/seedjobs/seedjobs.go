package seedjobs

import (
	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	jenkins "github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/client"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"

	"context"
	"errors"
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ConfigureSeedJobsName this is the fixed seed job name
	ConfigureSeedJobsName = "Configure Seed Jobs"

	deployKeyIDParameterName      = "DEPLOY_KEY_ID"
	privateKeyParameterName       = "PRIVATE_KEY"
	repositoryURLParameterName    = "REPOSITORY_URL"
	repositoryBranchParameterName = "REPOSITORY_BRANCH"
	targetsParameterName          = "TARGETS"
	displayNameParameterName      = "SEED_JOB_DISPLAY_NAME"
)

// ConfigureSeedJobs configures and triggers seed job pipeline for every Jenkins.Spec.SeedJobs entry
func ConfigureSeedJobs(jenkinsClient jenkins.Jenkins, k8sClient k8s.Client, jenkins *virtuslabv1alpha1.Jenkins) error {
	err := configureSeedJobsPipeline(jenkinsClient)
	if err != nil {
		return err
	}

	seedJobs := jenkins.Spec.SeedJobs
	for _, seedJob := range seedJobs {
		privateKey, err := extractPrivateKey(k8sClient, jenkins.Namespace, seedJob)
		if err != nil {
			return err
		}
		err = triggerConfigureSeedJobsPipeline(
			jenkinsClient,
			seedJob.ID,
			privateKey,
			seedJob.RepositoryURL,
			seedJob.RepositoryBranch, seedJob.Targets, fmt.Sprintf("Seed Job from %s", seedJob.ID))
		if err != nil {
			return err
		}
	}
	return nil
}

// configureSeedJobsPipeline configures seed jobs and deploy keys
func configureSeedJobsPipeline(jenkinsClient jenkins.Jenkins) error {
	return createOrUpdateSeedJob(jenkinsClient)
}

// FIXME this function should be part of jenkins client API
func createOrUpdateSeedJob(jenkinsClient jenkins.Jenkins) error {
	job, err := jenkinsClient.GetJob(ConfigureSeedJobsName)
	if jobNotExists(err) {
		_, err := jenkinsClient.CreateJob(seedJobConfigXML, ConfigureSeedJobsName)
		return err
	} else if err != nil {
		err := job.UpdateConfig(seedJobConfigXML)
		return err
	}
	return err
}

func jobNotExists(err error) bool {
	if err != nil {
		return err.Error() == errors.New("404").Error()
	}
	return false
}

// triggerConfigureSeedJobsPipeline triggers and configures seed job for specific GitHub repository
func triggerConfigureSeedJobsPipeline(jenkinsClient jenkins.Jenkins, deployKeyID, privateKey, repositoryURL, repositoryBranch, targets, displayName string) error {
	options := map[string]string{
		deployKeyIDParameterName:      deployKeyID,
		privateKeyParameterName:       privateKey,
		repositoryURLParameterName:    repositoryURL,
		repositoryBranchParameterName: repositoryBranch,
		targetsParameterName:          targets,
		displayNameParameterName:      displayName,
	}
	// FIXME implement EnsureJob()
	_, err := jenkinsClient.BuildJob(ConfigureSeedJobsName, options)
	if err != nil {
		return err
	}
	return nil
}

func extractPrivateKey(k8sClient k8s.Client, namespace string, seedJob virtuslabv1alpha1.SeedJob) (string, error) {
	if seedJob.PrivateKey.SecretKeyRef != nil {
		deployKeySecret := &v1.Secret{}
		namespaceName := types.NamespacedName{Namespace: namespace, Name: seedJob.PrivateKey.SecretKeyRef.Name}
		err := k8sClient.Get(context.TODO(), namespaceName, deployKeySecret)
		if err != nil {
			return "", err
		}
		return string(deployKeySecret.Data[seedJob.PrivateKey.SecretKeyRef.Key]), nil
	}
	return "", nil
}

// FIXME use mask-password plugin for params.PRIVATE_KEY
var seedJobConfigXML = `
<flow-definition plugin="workflow-job@2.30">
  <actions/>
  <description></description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <hudson.model.ParametersDefinitionProperty>
      <parameterDefinitions>
        <hudson.model.StringParameterDefinition>
          <name>DEPLOY_KEY_ID</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>PRIVATE_KEY</name>
          <description></description>
          <defaultValue></defaultValue>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>REPOSITORY_URL</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>REPOSITORY_BRANCH</name>
          <description></description>
          <defaultValue>master</defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>SEED_JOB_DISPLAY_NAME</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>TARGETS</name>
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


def jobDslSeedName = &quot;${params.DEPLOY_KEY_ID}-job-dsl-seed&quot;
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
GlobalConfiguration.all().get(GlobalJobDslSecurityConfiguration.class).save()</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
`
