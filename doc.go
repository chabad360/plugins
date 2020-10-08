// Package plugins allows you to load arbitrary Go code after compilation.
// Plugins are ran through Yaegi (https://github.com/treafik/yaegi), the Go interpreter.
//
// How it Works
//
// Plugins are downloaded as zip files and should be saved to a specified folder (e.g. /usr/lib/program/plugins).
// Once downloaded, the hash for the zip files are collected and stored.
// If there are any plugins in the cache folder, the hashes for their corresponding zip files are collected from the configuration (more on that later).
// The hashes are compared against each other, if there is a hash in the plugins cache that isn't in the list of zip hashes, that plugin is removed (plugins with the tag local set to true are ignored).
// If there is a hash for a zip file that isn't in the plugins cache, that plugin is unzipped.
//
// Once the plugins cache has been updated, the plugins are loaded by setting the GOPATH for the interpreter to the plugin folder and importing the plugin module as specifed by the import tag.
// The interpreter than retrieves the module.Plugin struct from the module.
// This struct should implement the plugin type as declared in the plugin configuration.
//
// Now that the plugins have been loaded, you can run GetPlugins() to get a list of all loaded plugins.
// Alternatively, you could run GetPluginsForType() to get a list of all loaded plugins that satisfy a certain plugin type.
// Once you know that, you can pass the name of the plugin you'd like to use to GetPlugin() which returns an interface that can be bound to the desired plugin type interface.
// From there you can use it however you'd like.
//
// Usage
//
// Usage is rather simple:
//  import "github.com/chabad360/plugins"
//
//  // GreeterPlugin is an example plugin interface
//  type GreeterPlugin interface {
//  	// Greet takes a string and outputs it somehow.
//  	Greet(string)
//  }
//
//  func main() {
//		pluginHost := plugins.NewPluginHost("./plugins", "./plugins-cache")
//		pluginHost.AddPluginType((*GreeterPlugin)(nil))
//		pluginType.LoadPlugins()
//
//		pluginI, err := pluginHost.GetPlugin("hello")
//		if err != nil {
//			panic(err)
//		}
//
//		This can be done safely, because during LoadPlugins(), we check to ensure that the plugins fulfill the type requirements.
//		plugin := pluginI.(GreeterPlugin)
//
//		plugin.Greet("Hello")
//		// Output: Hello
//  }
//
// Plugin Format
//
// Plugins are zip files with the following structure:
//  plugin.zip/ (name is arbitrary)
//   └ pluginFolder/ (name is arbitrary)
//      ├ plugin.yml
//      ├ vendor/ (required if the plugin has non-stdlib dependencies)
//      ├ go.mod
//      ├ main.go (name is arbitrary)
//      ┊
//
// plugin.yml example:
//  name: Plugin
//  description: This is a plugin that does plugin things.
//  import: github.com/user/plugin
//  type: middleware
//
package plugins
