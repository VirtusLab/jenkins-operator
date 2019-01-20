package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/bndr/gojenkins"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	assert "github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type amazonS3BackupConfiguration struct {
	BucketName string `json:"bucketName,omitempty"`
	BucketPath string `json:"bucketPath,omitempty"`
	Region     string `json:"region,omitempty"`
	AccessKey  string `json:"accessKey,omitempty"`
	SecretKey  string `json:"secretKey,omitempty"`
}

func TestAmazonS3Backup(t *testing.T) {
	t.Parallel()
	if amazonS3BackupConfigurationFile == nil || len(*amazonS3BackupConfigurationFile) == 0 {
		t.Skipf("Skipping testing because flag '%s' is not set", amazonS3BackupConfigurationParameterName)
	}
	backupConfig := loadAmazonS3BackupConfig(t)

	s3Client := createS3Client(t, backupConfig)
	deleteAllBackupsInS3(t, backupConfig, s3Client)
	namespace, ctx := setupTest(t)
	defer ctx.Cleanup() // Deletes test namespace

	jenkins := createJenkinsCRWithAmazonS3Backup(t, namespace, backupConfig)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	waitForJenkinsUserConfigurationToComplete(t, jenkins)

	restartJenkinsMasterPod(t, jenkins)
	waitForRecreateJenkinsMasterPod(t, jenkins)

	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	jenkinsClient := verifyJenkinsAPIConnection(t, jenkins)
	verifyIfBackupAndRestoreWasSuccessfull(t, jenkinsClient, backupConfig, s3Client)
}

func createS3Client(t *testing.T, backupConfig amazonS3BackupConfiguration) *s3.S3 {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(backupConfig.Region),
		Credentials: credentials.NewStaticCredentials(backupConfig.AccessKey, backupConfig.SecretKey, ""),
	})
	assert.NoError(t, err)

	return s3.New(sess)
}

func deleteAllBackupsInS3(t *testing.T, backupConfig amazonS3BackupConfiguration, s3Client *s3.S3) {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(backupConfig.BucketName),
		Key:    aws.String(backupConfig.BucketPath),
	}

	_, err := s3Client.DeleteObject(input)
	assert.NoError(t, err)
}

func verifyIfBackupAndRestoreWasSuccessfull(t *testing.T, jenkinsClient *gojenkins.Jenkins, backupConfig amazonS3BackupConfiguration, s3Client *s3.S3) {
	job, err := jenkinsClient.GetJob(constants.UserConfigurationJobName)
	assert.NoError(t, err)
	// jenkins runs twice(2) + 1 as next build number
	assert.Equal(t, int64(3), job.Raw.NextBuildNumber)

	listObjects, err := s3Client.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(backupConfig.BucketName),
		Marker: aws.String(backupConfig.BucketPath),
	})
	assert.NoError(t, err)
	t.Logf("Backups in S3:%+v", listObjects.Contents)
	assert.Equal(t, len(listObjects.Contents), 2)
	latestBackupFound := false
	for _, backup := range listObjects.Contents {
		if *backup.Key == fmt.Sprintf("%s/%s", backupConfig.BucketPath, constants.BackupLatestFileName) {
			latestBackupFound = true
		}
	}
	assert.True(t, latestBackupFound)
}

func createJenkinsCRWithAmazonS3Backup(t *testing.T, namespace string, backupConfig amazonS3BackupConfiguration) *virtuslabv1alpha1.Jenkins {
	jenkins := &virtuslabv1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e",
			Namespace: namespace,
		},
		Spec: virtuslabv1alpha1.JenkinsSpec{
			Backup: virtuslabv1alpha1.JenkinsBackupTypeAmazonS3,
			BackupAmazonS3: virtuslabv1alpha1.JenkinsBackupAmazonS3{
				Region:     backupConfig.Region,
				BucketPath: backupConfig.BucketPath,
				BucketName: backupConfig.BucketName,
			},
			Master: virtuslabv1alpha1.JenkinsMaster{
				Image: "jenkins/jenkins",
			},
		},
	}

	t.Logf("Jenkins CR %+v", *jenkins)
	err := framework.Global.Client.Create(context.TODO(), jenkins, nil)
	assert.NoError(t, err)

	backupCredentialsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GetBackupCredentialsSecretName(jenkins),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			constants.BackupAmazonS3SecretAccessKey: []byte(backupConfig.AccessKey),
			constants.BackupAmazonS3SecretSecretKey: []byte(backupConfig.SecretKey),
		},
	}
	err = framework.Global.Client.Create(context.TODO(), backupCredentialsSecret, nil)
	assert.NoError(t, err)

	return jenkins
}

func loadAmazonS3BackupConfig(t *testing.T) amazonS3BackupConfiguration {
	jsonFile, err := os.Open(*amazonS3BackupConfigurationFile)
	assert.NoError(t, err)
	defer func() { _ = jsonFile.Close() }()

	byteValue, err := ioutil.ReadAll(jsonFile)
	assert.NoError(t, err)

	var result amazonS3BackupConfiguration
	err = json.Unmarshal([]byte(byteValue), &result)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.AccessKey)
	assert.NotEmpty(t, result.BucketName)
	assert.NotEmpty(t, result.Region)
	assert.NotEmpty(t, result.SecretKey)
	result.BucketPath = t.Name()
	return result
}
