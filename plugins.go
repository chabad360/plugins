package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/xyproto/unzip"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// NewPluginHost initializes a PluginHost.
func NewPluginHost(pluginDir string, pluginCacheDir string) *PluginHost {
	host := &PluginHost{
		PluginDir:      pluginDir,
		PluginCacheDir: pluginCacheDir,
		Plugins:        make(map[string]plugin),
		PluginTypes:    make(map[string]reflect.Type),
	}

	return host
}

func (h *PluginHost) lazyInit() {
	if h.PluginDir == "" {
		h.PluginDir = "./plugins"
	}
	if h.PluginCacheDir == "" {
		h.PluginCacheDir = "./plugins-cache"
	}
	if h.Plugins == nil {
		h.Plugins = make(map[string]plugin)
	}
	if h.PluginTypes == nil {
		h.PluginTypes = make(map[string]reflect.Type)
	}
}

// AddPluginType adds a plugin type to the list.
// The interface for the pluginType parameter should be a nil of the plugin type interface:
//   (*PluginInterface)(nil)
func (h *PluginHost) AddPluginType(name string, pluginType interface{}) {
	h.lazyInit()
	h.PluginTypes[name] = reflect.TypeOf(pluginType).Elem()
}

// LoadPlugins loads up the plugins in the plugin directory.
func (h *PluginHost) LoadPlugins() error {
	pluginZips, plugins, err := h.loadPluginHashes()
	if err != nil {
		return err
	}

	// This needs to go here and not with the lower "for hash, plugin" loop, because otherwise we risk deleting a plugin after it's been updated.
	for hash, plugin := range plugins {
		if _, ok := pluginZips[hash]; !ok && !strings.HasPrefix(hash, "local") {
			if err = os.RemoveAll(strings.TrimSuffix(plugin, filepath.Base(plugin))); err != nil {
				return err
			}
		}
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
		p, err := loadPlugin(plugin, hash)
		if err != nil {
			return err
		}

		if err := h.validatePlugin(p.plugin, p.config.PluginType); err != nil {
			return err
		}

		h.Plugins[p.config.Name] = *p
	}

	return nil
}

func (h *PluginHost) loadPluginHashes() (map[string]string, map[string]string, error) {
	h.lazyInit()

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

		if config.Local {
			hash := sha256.New()
			if _, err := io.Copy(hash, f); err != nil {
				return err
			}
			pluginHashes[fmt.Sprintf("local-%x", hash.Sum(nil))] = path
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

// GetPlugin returns a plugin as an interface, provided you know what your getting, you can safely bind it to an interface.
func (h *PluginHost) GetPlugin(pluginName string) (interface{}, error) {
	if _, ok := h.Plugins[pluginName]; !ok {
		return nil, fmt.Errorf("no such plugin %s", pluginName)
	}

	return h.Plugins[pluginName].plugin, nil
}
