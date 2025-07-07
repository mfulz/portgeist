// Package configloader provides a generic runtime registry for configuration
// instances in Portgeist. It allows both the daemon and CLI to register and
// retrieve their specific configuration types in a type-safe, singleton manner.
//
// This mechanism enables shared access to configuration across packages (e.g. logging),
// without introducing cyclic dependencies or relying on global variables.
//
// Typical usage:
//
//	type Config struct { ... }
//	func init() {
//	    configloader.RegisterConfig(&Config{...})
//	}
//	cfg := configloader.MustGetConfig[*Config]()
package configloader

import (
	"fmt"
	"reflect"
	"sync"
)

var registry sync.Map // key = reflect.Type of the config type, value = registered config instance

// RegisterConfig registers a config instance of type T for global access.
//
// It panics if a config of the same type is already registered.
//
// Example:
//
//	RegisterConfig[*MyConfig](cfg)
func RegisterConfig[T any](cfg T) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if _, exists := registry.Load(t); exists {
		panic(fmt.Sprintf("config already registered for type %v", t))
	}
	registry.Store(t, cfg)
}

// MustGetConfig retrieves the registered config instance of type T.
//
// It panics if no config of type T has been registered.
//
// Example:
//
//	cfg := MustGetConfig[*MyConfig]()
func MustGetConfig[T any]() T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if val, ok := registry.Load(t); ok {
		return val.(T)
	}
	panic(fmt.Sprintf("no config registered for type %v", t))
}

// TryGetConfig retrieves the registered config instance of type T.
//
// It returns (zero-value, false) if the config was not found.
//
// Example:
//
//	if cfg, ok := TryGetConfig[*MyConfig](); ok {
//	    use(cfg)
//	}
func TryGetConfig[T any]() (T, bool) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if val, ok := registry.Load(t); ok {
		return val.(T), true
	}
	var zero T
	return zero, false
}
