package aws

import (
	"context"
	"fmt"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugins"
	"github.com/VirtusLab/jenkins-operator/pkg/log"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// AmazonS3Backup is a backup strategy where backup is stored in AWS S3 bucket
// credentials required to make calls to AWS API are provided by user in backup credentials Kubernetes secret
type AmazonS3Backup struct{}

// GetRestoreJobXML returns Jenkins restore backup job config XML
func (b *AmazonS3Backup) GetRestoreJobXML(jenkins virtuslabv1alpha1.Jenkins) (string, error) {
	return `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.31">
  <actions/>
  <description></description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <org.jenkinsci.plugins.workflow.job.properties.DisableConcurrentBuildsJobProperty/>
    <org.jenkinsci.plugins.workflow.job.properties.DisableResumeJobProperty/>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61.1">
    <script>import com.amazonaws.auth.PropertiesFileCredentialsProvider
import com.amazonaws.services.s3.AmazonS3ClientBuilder
import com.amazonaws.services.s3.model.AmazonS3Exception
import com.amazonaws.services.s3.model.S3Object

node(&apos;master&apos;) {
    def accessKeyFilePath = &quot;` + resources.JenkinsBackupCredentialsVolumePath + `/` + constants.BackupAmazonS3SecretAccessKey + `&quot;
    def secretKeyFilePath = &quot;` + resources.JenkinsBackupCredentialsVolumePath + `/` + constants.BackupAmazonS3SecretSecretKey + `&quot;
    def credentialsFileName = &quot;backup-credentials&quot;
    def bucketName = &quot;` + jenkins.Spec.BackupAmazonS3.BucketName + `&quot;
    def bucketKey = &quot;` + jenkins.Spec.BackupAmazonS3.BucketPath + `&quot;
    def region = &quot;` + jenkins.Spec.BackupAmazonS3.Region + `&quot;
    def latestBackupFile = &quot;` + constants.BackupLatestFileName + `&quot;

    def jenkinsHome = env.JENKINS_HOME
    def latestBackupKey = &quot;${bucketKey}/${latestBackupFile}&quot;
    def tmpBackupPath = &quot;/tmp/restore.tar.gz&quot;
    boolean backupExists = true

    def accessKey = new java.io.File(accessKeyFilePath).text
    def secretKey = new java.io.File(secretKeyFilePath).text
	sh &quot;touch ${env.WORKSPACE}/${credentialsFileName}&quot;
    new java.io.File(&quot;${env.WORKSPACE}/${credentialsFileName}&quot;).write(&quot;accessKey=${accessKey}\nsecretKey=${secretKey}\n&quot;)

    stage(&apos;Check if backup exists&apos;) {
        def s3 = AmazonS3ClientBuilder
                .standard()
                .withCredentials(new PropertiesFileCredentialsProvider(&quot;${env.WORKSPACE}/${credentialsFileName}&quot;))
                .withRegion(region)
                .build()
        try {
            println s3.getObjectMetadata(bucketName, latestBackupKey)
        } catch (AmazonS3Exception e) {
            if (e.getStatusCode() == 404) {
                println &quot;There is no backup ${bucketName}/${latestBackupKey}&quot;
                backupExists = false
            }
        }
    }

    if (backupExists) {
        stage(&apos;Download backup&apos;) {
            def s3 = AmazonS3ClientBuilder
                    .standard()
                    .withCredentials(new PropertiesFileCredentialsProvider(&quot;${env.WORKSPACE}/${credentialsFileName}&quot;))
                    .withRegion(region)
                    .build()
            S3Object backup = s3.getObject(bucketName, latestBackupKey)
            java.nio.file.Files.copy(
                    backup.getObjectContent(),
                    new java.io.File(tmpBackupPath).toPath(),
                    java.nio.file.StandardCopyOption.REPLACE_EXISTING);
        }

        stage(&apos;Unpack backup&apos;) {
            sh &quot;tar -C ${jenkinsHome} -zxf ${tmpBackupPath}&quot;
        }

        stage(&apos;Reload Jenkins&apos;) {
            jenkins.model.Jenkins.getInstance().reload()
        }

        sh &quot;rm ${tmpBackupPath}&quot;
		sh &quot;rm ${env.WORKSPACE}/${credentialsFileName}&quot;
    }
}</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>`, nil
}

// GetBackupJobXML returns Jenkins backup job config XML
func (b *AmazonS3Backup) GetBackupJobXML(jenkins virtuslabv1alpha1.Jenkins) (string, error) {
	return `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.31">
  <actions/>
  <description></description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <org.jenkinsci.plugins.workflow.job.properties.DisableConcurrentBuildsJobProperty/>
    <org.jenkinsci.plugins.workflow.job.properties.DisableResumeJobProperty/>
    <org.jenkinsci.plugins.workflow.job.properties.PipelineTriggersJobProperty>
      <triggers>
        <hudson.triggers.TimerTrigger>
          <spec>H/60 * * * *</spec>
        </hudson.triggers.TimerTrigger>
      </triggers>
    </org.jenkinsci.plugins.workflow.job.properties.PipelineTriggersJobProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61">
    <script>import com.amazonaws.auth.PropertiesFileCredentialsProvider
import com.amazonaws.services.s3.AmazonS3ClientBuilder

import java.io.File

node(&apos;master&apos;) {
    def accessKeyFilePath = &quot;` + resources.JenkinsBackupCredentialsVolumePath + `/` + constants.BackupAmazonS3SecretAccessKey + `&quot;
    def secretKeyFilePath = &quot;` + resources.JenkinsBackupCredentialsVolumePath + `/` + constants.BackupAmazonS3SecretSecretKey + `&quot;
    def credentialsFileName = &quot;backup-credentials&quot;
    def bucketName = &quot;` + jenkins.Spec.BackupAmazonS3.BucketName + `&quot;
    def bucketKey = &quot;` + jenkins.Spec.BackupAmazonS3.BucketPath + `&quot;
    def region = &quot;` + jenkins.Spec.BackupAmazonS3.Region + `&quot;
    def latestBackupFile = &quot;` + constants.BackupLatestFileName + `&quot;

    def jenkinsHome = env.JENKINS_HOME
    def backupTime = sh(script: &quot;date &apos;+%Y-%m-%d-%H-%M&apos;&quot;, returnStdout: true).trim()
    def tmpBackupPath = &quot;/tmp/backup.tar.gz&quot;

    def backupKey = &quot;${bucketKey}/build-history-${backupTime}.tar.gz&quot;
    def latestBackupKey = &quot;${bucketKey}/${latestBackupFile}&quot;

    def accessKey = new java.io.File(accessKeyFilePath).text
    def secretKey = new java.io.File(secretKeyFilePath).text
	sh &quot;touch ${env.WORKSPACE}/${credentialsFileName}&quot;
    new java.io.File(&quot;${env.WORKSPACE}/${credentialsFileName}&quot;).write(&quot;accessKey=${accessKey}\nsecretKey=${secretKey}\n&quot;)

    stage(&apos;Create backup archive&apos;) {
        println &quot;Creating backup archive to ${tmpBackupPath}&quot;
        sh &quot;tar -C ${jenkinsHome} -z --exclude jobs/*/config.xml --exclude jobs/*/workspace* --exclude jobs/*/simulation.log -c config-history jobs  -f ${tmpBackupPath}&quot;
    }

    stage(&apos;Upload backup&apos;) {
        def s3 = AmazonS3ClientBuilder
                .standard()
                .withCredentials(new PropertiesFileCredentialsProvider(&quot;${env.WORKSPACE}/${credentialsFileName}&quot;))
                .withRegion(region)
                .build()
        println &quot;Uploading backup to ${bucketName}/${backupKey}&quot;
        s3.putObject(bucketName, backupKey, new File(tmpBackupPath))
        println s3.getObjectMetadata(bucketName, backupKey)
    }

    stage(&apos;Copy backup&apos;) {
        def s3 = AmazonS3ClientBuilder
                .standard()
                .withCredentials(new PropertiesFileCredentialsProvider(&quot;${env.WORKSPACE}/${credentialsFileName}&quot;))
                .withRegion(region)
                .build()
        println &quot;Coping backup ${bucketName}${backupKey} to ${bucketName}/${latestBackupKey}&quot;
        s3.copyObject(bucketName, backupKey, bucketName, latestBackupKey)
        println s3.getObjectMetadata(bucketName, latestBackupKey)
    }

    sh &quot;rm ${tmpBackupPath}&quot;
	sh &quot;rm ${env.WORKSPACE}/${credentialsFileName}&quot;
}</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>`, nil
}

// IsConfigurationValidForBasePhase validates if user provided valid configuration of backup for base phase
func (b *AmazonS3Backup) IsConfigurationValidForBasePhase(jenkins virtuslabv1alpha1.Jenkins, logger logr.Logger) bool {
	if len(jenkins.Spec.BackupAmazonS3.BucketName) == 0 {
		logger.V(log.VWarn).Info("Bucket name not set in 'spec.backupAmazonS3.bucketName'")
		return false
	}

	if len(jenkins.Spec.BackupAmazonS3.BucketPath) == 0 {
		logger.V(log.VWarn).Info("Bucket path not set in 'spec.backupAmazonS3.bucketPath'")
		return false
	}

	if len(jenkins.Spec.BackupAmazonS3.Region) == 0 {
		logger.V(log.VWarn).Info("Region not set in 'spec.backupAmazonS3.region'")
		return false
	}

	return true
}

// IsConfigurationValidForUserPhase validates if user provided valid configuration of backup for user phase
func (b *AmazonS3Backup) IsConfigurationValidForUserPhase(k8sClient k8s.Client, jenkins virtuslabv1alpha1.Jenkins, logger logr.Logger) (bool, error) {
	backupSecretName := resources.GetBackupCredentialsSecretName(&jenkins)
	backupSecret := &corev1.Secret{}
	err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: jenkins.Namespace, Name: backupSecretName}, backupSecret)
	if err != nil {
		return false, err
	}

	if len(backupSecret.Data[constants.BackupAmazonS3SecretSecretKey]) == 0 {
		logger.V(log.VWarn).Info(fmt.Sprintf("Secret '%s' doesn't contains key: %s", backupSecretName, constants.BackupAmazonS3SecretSecretKey))
		return false, nil
	}

	if len(backupSecret.Data[constants.BackupAmazonS3SecretAccessKey]) == 0 {
		logger.V(log.VWarn).Info(fmt.Sprintf("Secret '%s' doesn't contains key: %s", backupSecretName, constants.BackupAmazonS3SecretAccessKey))
		return false, nil
	}

	return true, nil
}

// GetRequiredPlugins returns all required Jenkins plugins by this backup strategy
func (b *AmazonS3Backup) GetRequiredPlugins() map[string][]plugins.Plugin {
	return map[string][]plugins.Plugin{
		"aws-java-sdk:1.11.457": {
			plugins.Must(plugins.New(plugins.ApacheComponentsClientPlugin)),
			plugins.Must(plugins.New(plugins.Jackson2ADIPlugin)),
		},
	}
}
