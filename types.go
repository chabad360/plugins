package plugins

import (
	"errors"
	"reflect"
)

type PluginHost struct {
	PluginDir      string
	PluginCacheDir string

	Plugins     map[string]plugin
	PluginTypes map[string]reflect.Type
}

var (
	ErrLoading    = errors.New("loadPlugin")
	ErrValidating = errors.New("validatePlugin")
)
