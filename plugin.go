package plugins

import (
	"fmt"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

type plugin struct {
	config PluginConfig
	Path   string
	plugin interface{}
}

// PluginConfig Describes the configuration for the plugin.yml file to be found at the root of the plugin folder.
type PluginConfig struct {
	// ImportPath is the module path i.e. "github.com/user/module"
	ImportPath string `yaml:"import"`
	// PluginType is the type of plugin, this plugin is checked against that type.
	// The available types are specified by the implementor of this package.
	PluginType string `yaml:"type"`
	// Name is the of the plugin, it's used to identify the plugin.
	Name string `yaml:"name"`
	// Local determines if the plugin is sourced from a zip file or not.
	// This must be set to true if there is no matching zip file in the plugins folder, otherwise the plugin will be deleted.
	Local bool `yaml:"local,omitempty"`
	// Description is a purely aesthetic field to to fill with information about the plugin.
	Description string `yaml:"description"`
	// Hash is automatically filled by the plugins module. DO NOT TOUCH!!!
	// It is corresponding to the zip file that it came from.
	// If that zip file is missing (i.e. it's hash isn't in the plugins folder), it get's deleted.
	//
	// Again: DO NOT TOUCH!!!
	Hash string `yaml:"hash,omitempty"`
}

func (p *plugin) initPlugin() error {
	i := interp.New(interp.Options{GoPath: p.Path})
	i.Use(stdlib.Symbols)

	_, err := i.Eval(fmt.Sprintf(`import "%s"`, p.config.ImportPath))
	if err != nil {
		return err
	}

	v, err := i.Eval(filepath.Base(p.config.ImportPath) + ".Plugin")
	if err != nil {
		return err
	}

	p.plugin = v

	return nil
}

func loadPlugin(pluginPath string, hash string) (*plugin, error) {
	cF, err := os.Open(pluginPath)
	if err != nil {
		return nil, err
	}
	defer cF.Close()

	var c []byte
	if _, err := cF.Read(c); err != nil {
		return nil, err
	}

	var config PluginConfig
	if err := yaml.Unmarshal(c, &config); err != nil {
		return nil, err
	}

	if !config.Local && config.Hash == "" {
		config.Hash = hash

		c, err = yaml.Marshal(config)
		if err != nil {
			return nil, err
		}
		if _, err = cF.Write(c); err != nil {
			return nil, err
		}
	}

	p := plugin{
		config: config,
		Path:   strings.TrimSuffix(pluginPath, filepath.Base(pluginPath)),
	}

	if err := p.initPlugin(); err != nil {
		return nil, err
	}

	return &p, nil
}
