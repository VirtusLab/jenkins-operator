package nobackup

import (
	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/plugins"

	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// NoBackup is a backup strategy where there is no backup
type NoBackup struct{}

var emptyJob = `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.31">
  <actions/>
  <description></description>
  <keepDependencies>false</keepDependencies>
  <properties></properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61">
    <script></script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
`

// GetRestoreJobXML returns Jenkins restore backup job config XML
func (b *NoBackup) GetRestoreJobXML(jenkins virtuslabv1alpha1.Jenkins) (string, error) {
	return emptyJob, nil
}

// GetBackupJobXML returns Jenkins backup job config XML
func (b *NoBackup) GetBackupJobXML(jenkins virtuslabv1alpha1.Jenkins) (string, error) {
	return emptyJob, nil
}

// IsConfigurationValidForBasePhase validates if user provided valid configuration of backup for base phase
func (b *NoBackup) IsConfigurationValidForBasePhase(jenkins virtuslabv1alpha1.Jenkins, logger logr.Logger) bool {
	return true
}

// IsConfigurationValidForUserPhase validates if user provided valid configuration of backup for user phase
func (b *NoBackup) IsConfigurationValidForUserPhase(k8sClient k8s.Client, jenkins virtuslabv1alpha1.Jenkins, logger logr.Logger) (bool, error) {
	return true, nil
}

// GetRequiredPlugins returns all required Jenkins plugins by this backup strategy
func (b *NoBackup) GetRequiredPlugins() map[string][]plugins.Plugin {
	return map[string][]plugins.Plugin{}
}
