package constants

const (
	// OperatorName is a operator name
	OperatorName = "jenkins-operator"
	// DefaultAmountOfExecutors is the default amount of Jenkins executors
	DefaultAmountOfExecutors = 3
	// SeedJobSuffix is a suffix added for all seed jobs
	SeedJobSuffix = "job-dsl-seed"
	// DefaultJenkinsMasterImage is the default Jenkins master docker image
	DefaultJenkinsMasterImage = "jenkins/jenkins:lts"
	// BackupAmazonS3SecretAccessKey is the Amazon user access key used to Amazon S3 backup
	BackupAmazonS3SecretAccessKey = "access-key"
	// BackupAmazonS3SecretSecretKey is the Amazon user secret key used to Amazon S3 backup
	BackupAmazonS3SecretSecretKey = "secret-key"
	// BackupJobName is the Jenkins job name used to backup jobs history
	BackupJobName = OperatorName + "-backup"
	// UserConfigurationJobName is the Jenkins job name used to configure Jenkins by groovy scripts provided by user
	UserConfigurationJobName = OperatorName + "-user-configuration"
	// BackupLatestFileName is the latest backup file name
	BackupLatestFileName = "build-history-latest.tar.gz"
)
