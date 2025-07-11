// Package ilauncher defines extensible interfaces for launcher backend implementations.
// Each backend (e.g., proxychains, cgroups, torsocks) must implement LauncherBackend.
package ilauncher

import (
	"fmt"

	"github.com/spf13/cobra"
)

// FileConfig represents a launcher config from launchers/*.yaml
type FileConfig struct {
	Method         string            `yaml:"method"`          // backend type (e.g. "cgroup")
	Binary         string            `yaml:"binary"`          // Absolute path to wrapper binary
	Env            map[string]string `yaml:"env"`             // env vars for backend
	ConfigTemplate string            `yaml:"config_template"` // optional backend-specific config
}

// LauncherBackend represents a pluggable launch implementation.
type LauncherBackend interface {
	Method() string
	Build(name string, cfg FileConfig) (*cobra.Command, error)
}

// backendRegistry stores all registered backend types by method name.
var backendRegistry = map[string]LauncherBackend{}

// RegisterBackend adds a new launch backend to the registry.
func RegisterBackend(b LauncherBackend) {
	backendRegistry[b.Method()] = b
}

// GetBackend adds a new launch backend to the registry.
func GetBackend(n string) (LauncherBackend, error) {
	backend, ok := backendRegistry[n]
	if !ok {
		return nil, fmt.Errorf("unknown backend method: %s", n)
	}
	return backend, nil
}
