package plugins

const (
	// ApacheComponentsClientPlugin is apache-httpcomponents-client-4-api Jenkins plugin with version
	ApacheComponentsClientPlugin = "apache-httpcomponents-client-4-api:4.5.5-3.0"
	// Jackson2ADIPlugin is jackson2-api-httpcomponents-client-4-api Jenkins plugin with version
	Jackson2ADIPlugin = "jackson2-api:2.9.8"
)

// BasePluginsMap contains plugins to install by operator
var BasePluginsMap = map[string][]Plugin{
	Must(New("kubernetes:1.13.8")).String(): {
		Must(New(ApacheComponentsClientPlugin)),
		Must(New("cloudbees-folder:6.7")),
		Must(New("credentials:2.1.18")),
		Must(New("durable-task:1.28")),
		Must(New(Jackson2ADIPlugin)),
		Must(New("kubernetes-credentials:0.4.0")),
		Must(New("plain-credentials:1.5")),
		Must(New("structs:1.17")),
		Must(New("variant:1.1")),
		Must(New("workflow-step-api:2.17")),
	},
	Must(New("workflow-job:2.31")).String(): {
		Must(New("scm-api:2.3.0")),
		Must(New("script-security:1.50")),
		Must(New("structs:1.17")),
		Must(New("workflow-api:2.33")),
		Must(New("workflow-step-api:2.17")),
		Must(New("workflow-support:3.0")),
	},
	Must(New("workflow-aggregator:2.6")).String(): {
		Must(New("ace-editor:1.1")),
		Must(New(ApacheComponentsClientPlugin)),
		Must(New("authentication-tokens:1.3")),
		Must(New("branch-api:2.1.2")),
		Must(New("cloudbees-folder:6.7")),
		Must(New("credentials-binding:1.17")),
		Must(New("credentials:2.1.18")),
		Must(New("display-url-api:2.3.0")),
		Must(New("docker-commons:1.13")),
		Must(New("docker-workflow:1.17")),
		Must(New("durable-task:1.28")),
		Must(New("git-client:2.7.6")),
		Must(New("git-server:1.7")),
		Must(New("handlebars:1.1.1")),
		Must(New(Jackson2ADIPlugin)),
		Must(New("jquery-detached:1.2.1")),
		Must(New("jsch:0.1.55")),
		Must(New("junit:1.26.1")),
		Must(New("lockable-resources:2.3")),
		Must(New("mailer:1.23")),
		Must(New("matrix-project:1.13")),
		Must(New("momentjs:1.1.1")),
		Must(New("pipeline-build-step:2.7")),
		Must(New("pipeline-graph-analysis:1.9")),
		Must(New("pipeline-input-step:2.9")),
		Must(New("pipeline-milestone-step:1.3.1")),
		Must(New("pipeline-model-api:1.3.4.1")),
		Must(New("pipeline-model-declarative-agent:1.1.1")),
		Must(New("pipeline-model-definition:1.3.4.1")),
		Must(New("pipeline-model-extensions:1.3.4.1")),
		Must(New("pipeline-rest-api:2.10")),
		Must(New("pipeline-stage-step:2.3")),
		Must(New("pipeline-stage-tags-metadata:1.3.4.1")),
		Must(New("pipeline-stage-view:2.10")),
		Must(New("plain-credentials:1.5")),
		Must(New("scm-api:2.3.0")),
		Must(New("script-security:1.50")),
		Must(New("ssh-credentials:1.14")),
		Must(New("structs:1.17")),
		Must(New("workflow-api:2.33")),
		Must(New("workflow-basic-steps:2.13")),
		Must(New("workflow-cps-global-lib:2.12")),
		Must(New("workflow-cps:2.61.1")),
		Must(New("workflow-durable-task-step:2.27")),
		Must(New("workflow-job:2.31")),
		Must(New("workflow-multibranch:2.20")),
		Must(New("workflow-scm-step:2.7")),
		Must(New("workflow-step-api:2.17")),
		Must(New("workflow-support:3.0")),
	},
	Must(New("git:3.9.1")).String(): {
		Must(New(ApacheComponentsClientPlugin)),
		Must(New("credentials:2.1.18")),
		Must(New("display-url-api:2.3.0")),
		Must(New("git-client:2.7.6")),
		Must(New("jsch:0.1.55")),
		Must(New("junit:1.26.1")),
		Must(New("mailer:1.23")),
		Must(New("matrix-project:1.13")),
		Must(New("scm-api:2.3.0")),
		Must(New("script-security:1.50")),
		Must(New("ssh-credentials:1.14")),
		Must(New("structs:1.17")),
		Must(New("workflow-api:2.33")),
		Must(New("workflow-scm-step:2.7")),
		Must(New("workflow-step-api:2.17")),
	},
	Must(New("job-dsl:1.71")).String(): {
		Must(New("script-security:1.50")),
		Must(New("structs:1.17")),
	},
	Must(New("jobConfigHistory:2.19")).String(): {},
	Must(New("configuration-as-code:1.4")).String(): {
		Must(New("configuration-as-code-support:1.4")),
	},
	Must(New("simple-theme-plugin:0.5.1")).String(): {},
}

// BasePlugins returns map of plugins to install by operator
func BasePlugins() (plugins map[string][]string) {
	plugins = map[string][]string{}

	for rootPluginName, dependentPlugins := range BasePluginsMap {
		plugins[rootPluginName] = []string{}
		for _, pluginName := range dependentPlugins {
			plugins[rootPluginName] = append(plugins[rootPluginName], pluginName.String())
		}
	}

	return
}
