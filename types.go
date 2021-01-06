package plugins

import (
	"errors"
	"reflect"
)

// PluginHost manages loading the plugins.
type PluginHost struct {
	// PluginDir is where the zipped plugins are stored.
	PluginDir      string
	// PluginCacheDir is the location where the plugins are unzipped too.
	PluginCacheDir string
	// Plugins contains a map of the plugins, you can run operations directly on this.
	Plugins     map[string]plugin
	// PluginTypes is a list of plugins types that plugins have to use at least one of.
	PluginTypes map[string]reflect.Type
}

var (
	// ErrLoading is returned when there is an issue loading the plugin files.
	ErrLoading    = errors.New("error loading plugin")
	// ErrInvalidType is returned when the plugin type specified by the plugin is invalid.
	ErrInvalidType = errors.New("invalid plugin type")
	// ErrValidatingPlugin is returned when the plugin fails to fully implement the interface of the plugin type.
	ErrValidatingPlugin = errors.New("plugin does not implement type")
)
