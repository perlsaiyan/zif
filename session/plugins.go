package session

import (
	"plugin"

	"github.com/perlsaiyan/zif/config"
)

type PluginRegistry struct {
	Plugin *plugin.Plugin
	Active bool
}

type PluginInfo struct {
	Name    string
	Version string
}

func LoadPlugin(path string, config *config.Config) (*plugin.Plugin, error) {

	// In order to load a plugin, do something like this:

	plugin, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	return plugin, nil
}
