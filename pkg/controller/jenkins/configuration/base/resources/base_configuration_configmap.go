package resources

import (
	"fmt"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const basicSettingsFmt = `
import jenkins.model.Jenkins
import jenkins.model.JenkinsLocationConfiguration
import hudson.model.Node.Mode

def jenkins = Jenkins.instance
//Number of jobs that run simultaneously on master, currently only backup and SeedJob.
jenkins.setNumExecutors(%d)
//Jobs must specify that they want to run on master
jenkins.setMode(Mode.EXCLUSIVE)
jenkins.save()

`

const enableCSRF = `
import hudson.security.csrf.DefaultCrumbIssuer
import jenkins.model.Jenkins

def jenkins = Jenkins.instance

if (jenkins.getCrumbIssuer() == null) {
    jenkins.setCrumbIssuer(new DefaultCrumbIssuer(true))
    jenkins.save()
    println('CSRF Protection enabled.')
} else {
    println('CSRF Protection already configured.')
}
`

const disableUsageStats = `
import jenkins.model.Jenkins

def jenkins = Jenkins.instance

if (jenkins.isUsageStatisticsCollected()) {
    jenkins.setNoUsageStatistics(true)
    jenkins.save()
    println('Jenkins usage stats submitting disabled.')
} else {
    println('Nothing changed.  Usage stats are not submitted to the Jenkins project.')
}
`

const enableMasterAccessControl = `
import jenkins.security.s2m.AdminWhitelistRule
import jenkins.model.Jenkins

// see https://wiki.jenkins-ci.org/display/JENKINS/Slave+To+Master+Access+Control
def jenkins = Jenkins.instance
jenkins.getInjector()
        .getInstance(AdminWhitelistRule.class)
        .setMasterKillSwitch(false) // for real though, false equals enabled..........
jenkins.save()
`

const disableInsecureFeatures = `
import jenkins.*
import jenkins.model.*
import hudson.model.*
import jenkins.security.s2m.*

def jenkins = Jenkins.instance

println("Disabling insecure Jenkins features...")

println("Disabling insecure protocols...")
println("Old protocols: [" + jenkins.getAgentProtocols().join(", ") + "]")
HashSet<String> newProtocols = new HashSet<>(jenkins.getAgentProtocols())
newProtocols.removeAll(Arrays.asList("JNLP3-connect", "JNLP2-connect", "JNLP-connect", "CLI-connect"))
println("New protocols: [" + newProtocols.join(", ") + "]")
jenkins.setAgentProtocols(newProtocols)

println("Disabling CLI access of /cli URL...")
def remove = { list ->
    list.each { item ->
        if (item.getClass().name.contains("CLIAction")) {
            println("Removing extension ${item.getClass().name}")
            list.remove(item)
        }
    }
}
remove(jenkins.getExtensionList(RootAction.class))
remove(jenkins.actions)

println("Disable CLI completely...")
CLI.get().setEnabled(false)
println("CLI disabled")

jenkins.save()
`

const configureKubernetesPluginFmt = `
import com.cloudbees.plugins.credentials.CredentialsScope
import com.cloudbees.plugins.credentials.SystemCredentialsProvider
import com.cloudbees.plugins.credentials.domains.Domain
import jenkins.model.Jenkins
import org.csanchez.jenkins.plugins.kubernetes.KubernetesCloud
import org.csanchez.jenkins.plugins.kubernetes.ServiceAccountCredential

def kubernetesCredentialsId = 'kubernetes-namespace-token'
def jenkins = Jenkins.getInstance()

ServiceAccountCredential serviceAccountCredential = new ServiceAccountCredential(
        CredentialsScope.GLOBAL,
        kubernetesCredentialsId,
        "Kubernetes Namespace Token"
)
SystemCredentialsProvider.getInstance().getStore().addCredentials(Domain.global(), serviceAccountCredential)

KubernetesCloud kubernetes = new KubernetesCloud("kubernetes")
kubernetes.setServerUrl("https://kubernetes.default")
kubernetes.setNamespace("%s")
kubernetes.setCredentialsId(kubernetesCredentialsId)
kubernetes.setJenkinsUrl("http://%s:%d")
kubernetes.setRetentionTimeout(15)
jenkins.clouds.add(kubernetes)

jenkins.save()
`

const configureViews = `
import hudson.model.ListView
import jenkins.model.Jenkins

def Jenkins jenkins = Jenkins.getInstance()

def seedViewName = 'seed-jobs'
def nonSeedViewName = 'non-seed-jobs'
def jenkinsViewName = '` + constants.OperatorName + `'

if (jenkins.getView(seedViewName) == null) {
    def seedView = new ListView(seedViewName)
    seedView.setIncludeRegex('.*` + constants.SeedJobSuffix + `.*')
    jenkins.addView(seedView)
}

if (jenkins.getView(nonSeedViewName) == null) {
    def nonSeedView = new ListView(nonSeedViewName)
    nonSeedView.setIncludeRegex('((?!seed)(?!jenkins).)*')
    jenkins.addView(nonSeedView)
}

if (jenkins.getView(jenkinsViewName) == null) {
    def jenkinsView = new ListView(jenkinsViewName)
    jenkinsView.setIncludeRegex('.*` + constants.OperatorName + `.*')
    jenkins.addView(jenkinsView)
}

jenkins.save()
`

// GetBaseConfigurationConfigMapName returns name of Kubernetes config map used to base configuration
func GetBaseConfigurationConfigMapName(jenkins *virtuslabv1alpha1.Jenkins) string {
	return fmt.Sprintf("%s-base-configuration-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}

// NewBaseConfigurationConfigMap builds Kubernetes config map used to base configuration
func NewBaseConfigurationConfigMap(meta metav1.ObjectMeta, jenkins *virtuslabv1alpha1.Jenkins) (*corev1.ConfigMap, error) {
	meta.Name = GetBaseConfigurationConfigMapName(jenkins)

	return &corev1.ConfigMap{
		TypeMeta:   buildConfigMapTypeMeta(),
		ObjectMeta: meta,
		Data: map[string]string{
			"1-basic-settings.groovy":               fmt.Sprintf(basicSettingsFmt, constants.DefaultAmountOfExecutors),
			"2-enable-csrf.groovy":                  enableCSRF,
			"3-disable-usage-stats.groovy":          disableUsageStats,
			"4-enable-master-access-control.groovy": enableMasterAccessControl,
			"5-disable-insecure-features.groovy":    disableInsecureFeatures,
			"6-configure-kubernetes-plugin.groovy": fmt.Sprintf(configureKubernetesPluginFmt,
				jenkins.ObjectMeta.Namespace, GetResourceName(jenkins), HTTPPortInt),
			"7-configure-views.groovy": configureViews,
		},
	}, nil
}
