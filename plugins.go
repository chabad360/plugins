package plugins

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"path/filepath"
	"reflect"
)

type Plugin struct {
	config  PluginConfig
	Path    string
	ZipHash []byte
	p       interface{}
}

type PluginConfig struct {
	ImportPath string
	PluginType string
	Name       string
}

type PluginHost struct {
	PluginDir      string
	PluginCacheDir string

	Plugins     map[string]Plugin
	PluginTypes map[string]reflect.Type
}

var (
	LoadingError    = errors.New("loadPlugins")
	ValidatingError = errors.New("validatePlugin")
)

// NewPluginHost initializes a PluginHost.
func NewPluginHost(pluginDir string, pluginCacheDir string) *PluginHost {
	host := &PluginHost{
		PluginDir:      pluginDir,
		PluginCacheDir: pluginCacheDir,
		Plugins:        make(map[string]Plugin),
		PluginTypes:    make(map[string]reflect.Type),
	}

	return host
}

// AddPluginType adds a new interface to the Plugin Host. The pluginType parameter should look like this: (*Interface)(nil).
func (h *PluginHost) AddPluginType(name string, pluginType interface{}) {
	h.PluginTypes[name] = reflect.TypeOf(pluginType).Elem()
}

func (h *PluginHost) LoadPlugins() error {
	return filepath.Walk(h.PluginDir, h.loadPlugin)
}

func (h *PluginHost) loadPlugin(path string, info os.FileInfo, err error) error {
	if filepath.Ext(path) != ".zip" {
		return fmt.Errorf("%w: File %v is not a zip file", LoadingError, path)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	zipHash := hash.Sum(nil)

	//
	// Unzip the plugin
	//
	extractedFolder := ""

	pluginPath := filepath.Join(h.PluginCacheDir, extractedFolder)

	cF, err := os.Open(filepath.Join(pluginPath, "plugin.yml"))
	if err != nil {
		return err
	}
	defer cF.Close()

	var c []byte
	if _, err := cF.Read(c); err != nil {
		return err
	}

	var config PluginConfig
	if err := yaml.Unmarshal(c, &config); err != nil {
		return err
	}

	v, err := h.initPlugin(config, pluginPath, err)
	if err != nil {
		return err
	}

	h.Plugins[config.Name] = Plugin{
		config:  config,
		ZipHash: zipHash,
		Path:    pluginPath,
		p:       v,
	}

	return nil
}

func (h *PluginHost) initPlugin(config PluginConfig, extractedPath string, err error) (interface{}, error) {
	i := interp.New(interp.Options{GoPath: extractedPath})
	i.Use(stdlib.Symbols)

	_, err = i.Eval(fmt.Sprintf(`import "%s"`, config.ImportPath))
	if err != nil {
		return nil, err
	}

	v, err := i.Eval(filepath.Base(config.ImportPath) + ".Plugin")
	if err != nil {
		return nil, err
	}

	if err := h.validatePlugin(v, config.PluginType); err != nil {
		return nil, err
	}

	return v, nil
}

func (h *PluginHost) validatePlugin(p interface{}, pluginType string) error {
	pType := reflect.TypeOf(p)

	if _, ok := h.PluginTypes[pluginType]; !ok {
		return fmt.Errorf("%w: Plugin type %v is not a valid plugin type.", ValidatingError, pluginType)
	}

	if ok := pType.Implements(h.PluginTypes[pluginType]); !ok {
		return fmt.Errorf("%w: Plugin %v does not implement the %v plugin type.", ValidatingError, p, pluginType)
	}

	return nil
}

func (h *PluginHost) GetPlugin(pluginName string) (interface{}, error) {
	if _, ok := h.Plugins[pluginName]; !ok {
		return nil, fmt.Errorf("No such plugin %s", pluginName)
	}

	return h.Plugins[pluginName].p, nil
}
