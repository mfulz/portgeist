// Package launchcli provides dynamic configuration and helpers
// for launching external processes via proxy wrappers (e.g. proxychains, torsocks).
package launchcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mfulz/portgeist/interfaces/ilauncher"
	"github.com/mfulz/portgeist/internal/configloader"
	"gopkg.in/yaml.v3"
)

// FileConfig defines a YAML-backed launcher configuration.
// It is loaded from ~/.portgeist/geistctl/launch.yaml or /etc/portgeist/launch.yaml.
type LaunchConfig struct {
	Default   string `yaml:"default"` // Launcher method name (e.g. proxychains)
	Launchers map[string]*ilauncher.FileConfig
}

// loadLauncherConfigs scans launchers/*.yaml and builds CLI commands.
func loadLauncherConfigs(dir string) (map[string]*ilauncher.FileConfig, error) {
	backends := make(map[string]*ilauncher.FileConfig)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read launchers dir: %w", err)
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.Name(), err)
		}

		var fc ilauncher.FileConfig
		if err := yaml.Unmarshal(data, &fc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f.Name(), err)
		}

		name := strings.TrimSuffix(f.Name(), ".yaml")
		backends[name] = &fc
	}

	return backends, nil
}

// LoadLauncherConfig loads the launch configuration file using the configloader.
// The file is expected at ~/.portgeist/geistctl/launch.yaml or /etc/portgeist/launch.yaml.
func LoadLauncherConfig() (*LaunchConfig, error) {
	path, err := configloader.ResolveConfigPath("geistctl", "launch.yaml")
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	launcherPath := filepath.Join(filepath.Dir(path), "launchers")
	launchers, err := loadLauncherConfigs(launcherPath)
	if err != nil {
		return nil, err
	}

	var cfg LaunchConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	cfg.Launchers = launchers

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
