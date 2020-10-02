package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/xyproto/unzip"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// NewPluginHost initializes a PluginHost. The interfaces in the the map should be a nil of the plugin type interface: (*PluginInterface)(nil) .
func NewPluginHost(pluginDir string, pluginCacheDir string, pluginTypes map[string]interface{}) *PluginHost {
	host := &PluginHost{
		PluginDir:      pluginDir,
		PluginCacheDir: pluginCacheDir,
		Plugins:        make(map[string]Plugin),
		PluginTypes:    make(map[string]reflect.Type),
	}

	for name, pluginType := range pluginTypes {
		host.PluginTypes[name] = reflect.TypeOf(pluginType).Elem()
	}

	return host
}

func (h *PluginHost) LoadPlugins() error {
	pluginZips, plugins, err := h.loadPluginHashes()
	if err != nil {
		return err
	}

	for hash, zip := range pluginZips {
		if _, ok := plugins[hash]; ok {
			continue
		}

		if err := unzip.Extract(zip, h.PluginCacheDir); err != nil {
			return err
		}
		pluginPath := filepath.Join(h.PluginCacheDir, strings.TrimSuffix(filepath.Base(zip), ".zip"))

		plugins[hash] = filepath.Join(pluginPath, "plugin.yml")
	}

	for hash, plugin := range plugins {
		err = h.loadPlugin(plugin, hash)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *PluginHost) loadPlugin(plugin string, hash string) error {
	cF, err := os.Open(plugin)
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

	if !config.Local {
		config.Hash = hash

		c, err = yaml.Marshal(config)
		if err != nil {
			return err
		}
		if _, err = cF.Write(c); err != nil {
			return err
		}
	}

	pluginPath := strings.TrimSuffix(plugin, filepath.Base(plugin))

	v, err := h.initPlugin(config, pluginPath)
	if err != nil {
		return err
	}

	h.Plugins[config.Name] = Plugin{
		config: config,
		Path:   pluginPath,
		p:      v,
	}

	return nil
}

func (h *PluginHost) loadPluginHashes() (map[string]string, map[string]string, error) {
	zipHashes := make(map[string]string)
	pluginHashes := make(map[string]string)

	err := filepath.Walk(h.PluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != ".zip" {
			return fmt.Errorf("%w: File %v is not a zip file", ErrLoading, path)
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
		zipHashes[hex.EncodeToString(hash.Sum(nil))] = path

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	err = filepath.Walk(h.PluginCacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) != "pluigin.yml" {
			return filepath.SkipDir
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		var c []byte
		if _, err := f.Read(c); err != nil {
			return err
		}

		var config PluginConfig
		if err := yaml.Unmarshal(c, &config); err != nil {
			return err
		}

		if config.Local && config.Hash != "" {
			return nil
		}

		if _, ok := zipHashes[config.Hash]; !ok {
			if err := os.RemoveAll(strings.TrimSuffix(path, filepath.Base(path))); err != nil {
				return err
			}
		} else {
			pluginHashes[config.Hash] = path
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return zipHashes, pluginHashes, nil
}

func (h *PluginHost) initPlugin(config PluginConfig, extractedPath string) (interface{}, error) {
	i := interp.New(interp.Options{GoPath: extractedPath})
	i.Use(stdlib.Symbols)

	_, err := i.Eval(fmt.Sprintf(`import "%s"`, config.ImportPath))
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
		return fmt.Errorf("%w: Plugin type %v is not a valid plugin type", ErrValidating, pluginType)
	}

	if ok := pType.Implements(h.PluginTypes[pluginType]); !ok {
		return fmt.Errorf("%w: Plugin %v does not implement the %v plugin type", ErrValidating, p, pluginType)
	}

	return nil
}

func (h *PluginHost) GetPlugin(pluginName string) (interface{}, error) {
	if _, ok := h.Plugins[pluginName]; !ok {
		return nil, fmt.Errorf("no such plugin %s", pluginName)
	}

	return h.Plugins[pluginName].p, nil
}
