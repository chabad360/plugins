package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/xyproto/unzip"
)

// NewPluginHost initializes a PluginHost.
func NewPluginHost(pluginDir string, pluginCacheDir string, symbols map[string]map[string]reflect.Value) (*PluginHost, error) {
	if _, err := os.Stat(pluginDir); pluginDir != "" && os.IsNotExist(err) {
		return nil, fmt.Errorf("NewPluginHost: %w", err)
	}
	if _, err := os.Stat(pluginCacheDir); pluginCacheDir != "" && os.IsNotExist(err) {
		return nil, fmt.Errorf("NewPluginHost: %w", err)
	}

	host := &PluginHost{
		PluginDir:      pluginDir,
		PluginCacheDir: pluginCacheDir,
		Plugins:        make(map[string]plugin),
		PluginTypes:    make(map[string]reflect.Type),
		Symbols: symbols,
	}

	return host, nil
}

func (h *PluginHost) lazyInit() { // TODO: make this use sync.Once
	if h.Plugins == nil {
		h.Plugins = make(map[string]plugin)
	}
	if h.PluginTypes == nil {
		h.PluginTypes = make(map[string]reflect.Type)
	}
}

// AddPluginType adds a plugin type to the list.
// The interface for the pluginType parameter should be a nil of the plugin type interface:
//  (*PluginInterface)(nil)
func (h *PluginHost) AddPluginType(name string, pluginType interface{}) {
	h.lazyInit()
	h.PluginTypes[name] = reflect.TypeOf(pluginType).Elem()
}

// LoadPlugins loads up the plugins in the plugin directory.
func (h *PluginHost) LoadPlugins() error {
	if h.PluginDir == "" || h.PluginCacheDir == "" {
		return fmt.Errorf("LoadPlugins: %w", ErrNoDirectorySpecified)
	}

	pluginZips, plugins, err := h.loadPluginHashes()
	if err != nil {
		return err
	}

	// This needs to go here and not with the lower "for hash, plugin" loop, because otherwise we risk deleting a plugin after it's been updated.
	for hash, plugin := range plugins {
		if _, ok := pluginZips[hash]; !ok && !strings.HasPrefix(hash, "local") {
			fmt.Println(hash)
			fmt.Println(plugin)
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
		p, err := loadPlugin(plugin, hash, *h)
		if err != nil {
			return err
		}

		if err := h.validatePlugin(*p); err != nil {
			return err
		}

		h.Plugins[p.Config.Name] = *p
	}

	return nil
}

// AddInternalPlugin adds an compiled plugin as a valid plugin is the plugin system.
func (h *PluginHost) AddInternalPlugin(v reflect.Value, conf PluginConfig) error {
	conf.Internal = true

	p := plugin{
		Config: conf,
		Path:   "internal/" + conf.Name,
		plugin: v,
	}

	if err := h.validatePlugin(p); err != nil {
		return err
	}

	h.Plugins[conf.Name] = p

	return nil
}

func (h *PluginHost) loadPluginHashes() (map[string]string, map[string]string, error) {
	h.lazyInit()

	zipHashes := make(map[string]string)
	pluginHashes := make(map[string]string)

	err := filepath.Walk(h.PluginDir, walkZipHashes(zipHashes))
	if err != nil {
		return nil, nil, err
	}

	err = filepath.Walk(h.PluginCacheDir, walkPluginHashes(pluginHashes))
	if err != nil {
		return nil, nil, err
	}

	return zipHashes, pluginHashes, nil
}

func (h *PluginHost) validatePlugin(p plugin) error {
	pType := reflect.TypeOf(p.plugin.Interface())

	if _, ok := h.PluginTypes[p.Config.PluginType]; !ok {
		return fmt.Errorf("validatePlugin: %v: %w", p.Config.PluginType, ErrInvalidType)
	}

	if !pType.Implements(h.PluginTypes[p.Config.PluginType]){
		return fmt.Errorf("validatePlugin:%v: %w %v", p, ErrValidatingPlugin, p.Config.PluginType)
	}

	return nil
}

// GetPlugins returns a list of the plugins by name.
func (h *PluginHost) GetPlugins() (list []string) {
	for k := range h.Plugins {
		list = append(list, k)
	}

	return
}

// GetPluginsForType returns all the plugins that are of type pluginType or empty if the pluginType doesn't exist.
func (h *PluginHost) GetPluginsForType(pluginType string) (list []string) {
	if _, ok := h.PluginTypes[pluginType]; !ok {
		return
	}

	for k, v := range h.Plugins {
		if v.Config.PluginType == pluginType {
			list = append(list, k)
		}
	}

	return
}

// GetPlugin returns a plugin as an reflect.Value, provided you know what your getting, you can safely bind the value.Interface() to an interface.
func (h *PluginHost) GetPlugin(pluginName string) (reflect.Value, bool) {
	if _, ok := h.Plugins[pluginName]; !ok {
		return reflect.ValueOf(nil), false
	}

	return h.Plugins[pluginName].plugin, true
}
