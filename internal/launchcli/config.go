// Package launchcli provides dynamic configuration and helpers
// for launching external processes via proxy wrappers (e.g. proxychains, torsocks).
package launchcli

import (
	"fmt"
	"os"
	"strings"

	"github.com/mfulz/portgeist/internal/configloader"
	"gopkg.in/yaml.v3"
)

// FileConfig defines a YAML-backed launcher configuration.
// It is loaded from ~/.portgeist/geistctl/launch.yaml or /etc/portgeist/launch.yaml.
type FileConfig struct {
	Method         string            `yaml:"method"`          // Launcher method name (e.g. proxychains)
	Binary         string            `yaml:"binary"`          // Absolute path to wrapper binary
	Env            map[string]string `yaml:"env"`             // Optional environment variables
	ConfigTemplate string            `yaml:"config_template"` // Proxy config content with {{PORT}} placeholder
}

// LoadFileConfig loads the launch configuration file using the configloader.
// The file is expected at ~/.portgeist/geistctl/launch.yaml or /etc/portgeist/launch.yaml.
func LoadFileConfig() (*FileConfig, error) {
	path, err := configloader.ResolveConfigPath("geistctl", "launch.yaml")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg FileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	return &cfg, nil
}

// GenerateProxyConf takes a config template string and replaces the {{PORT}} placeholder.
// It writes the result to a temporary file and returns the file path.
func GenerateProxyConf(template string, port int) (string, error) {
	content := strings.ReplaceAll(template, "{{PORT}}", fmt.Sprintf("%d", port))
	tmpfile, err := os.CreateTemp("", "proxywrap_*.conf")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	_, err = tmpfile.WriteString(content)
	return tmpfile.Name(), err
}
