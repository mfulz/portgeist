// Package controlcli handles loading and managing local geistctl configuration.
// This includes user tokens and known daemon connection targets.
package controlcli

import (
	"fmt"
	"os"

	"github.com/mfulz/portgeist/internal/configloader"
	"gopkg.in/yaml.v3"
)

// UserConfig represents authentication info for a specific logical user.
type UserConfig struct {
	Token string `yaml:"token"`
}

// DaemonConfig represents one connection target (unix socket or TCP).
type DaemonConfig struct {
	Socket string `yaml:"socket,omitempty"`
	TCP    string `yaml:"tcp,omitempty"`
}

// CTLConfig holds the entire client-side geistctl configuration.
type CTLConfig struct {
	Users   map[string]UserConfig   `yaml:"users"`
	Daemons map[string]DaemonConfig `yaml:"daemons"`
}

// LoadCTLConfig loads configuration from ~/.portgeist or /etc/portgeist.
func LoadCTLConfig() (*CTLConfig, error) {
	cfgPath, err := configloader.ResolveConfigPath("geistctl", "config.yaml")
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config CTLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}
