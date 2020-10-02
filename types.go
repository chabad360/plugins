package plugins

import (
	"errors"
	"reflect"
)

type Plugin struct {
	config PluginConfig
	Path   string
	p      interface{}
}

type PluginConfig struct {
	ImportPath  string `yaml:"import"`
	PluginType  string `yaml:"type"`
	Name        string `yaml:"name"`
	Local       bool   `yaml:"local,omitempty"`
	Description string `yaml:"description"`
	Hash        string `yaml:"hash,omitempty"`
}

type PluginHost struct {
	PluginDir      string
	PluginCacheDir string

	Plugins     map[string]Plugin
	PluginTypes map[string]reflect.Type
}

var (
	ErrLoading    = errors.New("loadPlugin")
	ErrValidating = errors.New("validatePlugin")
)
