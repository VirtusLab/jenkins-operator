package plugins

import (
	"fmt"
	"strings"

	"github.com/VirtusLab/jenkins-operator/pkg/log"
)

// Plugin represents jenkins plugin
type Plugin struct {
	Name                     string `json:"name"`
	Version                  string `json:"version"`
	rootPluginNameAndVersion string
}

func (p Plugin) String() string {
	return fmt.Sprintf("%s:%s", p.Name, p.Version)
}

// New creates plugin from string, for example "name-of-plugin:0.0.1"
func New(nameWithVersion string) (*Plugin, error) {
	val := strings.SplitN(nameWithVersion, ":", 2)
	if val == nil || len(val) != 2 {
		return nil, fmt.Errorf("invalid plugin format '%s'", nameWithVersion)
	}
	return &Plugin{
		Name:    val[0],
		Version: val[1],
	}, nil
}

// Must returns plugin from pointer and throws panic when error is set
func Must(plugin *Plugin, err error) Plugin {
	if err != nil {
		panic(err)
	}

	return *plugin
}

// VerifyDependencies checks if all plugins have compatible versions
func VerifyDependencies(values ...map[string][]Plugin) bool {
	// key - plugin name, value array of versions
	allPlugins := make(map[string][]Plugin)
	valid := true

	for _, value := range values {
		for rootPluginNameAndVersion, plugins := range value {
			if rootPlugin, err := New(rootPluginNameAndVersion); err != nil {
				valid = false
			} else {
				allPlugins[rootPlugin.Name] = append(allPlugins[rootPlugin.Name], Plugin{
					Name:                     rootPlugin.Name,
					Version:                  rootPlugin.Version,
					rootPluginNameAndVersion: rootPluginNameAndVersion})
			}
			for _, plugin := range plugins {
				allPlugins[plugin.Name] = append(allPlugins[plugin.Name], Plugin{
					Name:                     plugin.Name,
					Version:                  plugin.Version,
					rootPluginNameAndVersion: rootPluginNameAndVersion})
			}
		}
	}

	for pluginName, versions := range allPlugins {
		if len(versions) == 1 {
			continue
		}

		for _, firstVersion := range versions {
			for _, secondVersion := range versions {
				if firstVersion.Version != secondVersion.Version {
					log.Log.V(log.VWarn).Info(fmt.Sprintf("Plugin '%s' requires version '%s' but plugin '%s' requires '%s' for plugin '%s'",
						firstVersion.rootPluginNameAndVersion,
						firstVersion.Version,
						secondVersion.rootPluginNameAndVersion,
						secondVersion.Version,
						pluginName,
					))
					valid = false
				}
			}
		}
	}

	return valid
}
