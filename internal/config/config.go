// Package config provides loading and parsing of the Portgeist configuration
// file using Viper. It defines the full configuration schema and exposes
// functions to access it at runtime.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the full structure of the portgeist configuration file.
type Config struct {
	Logins  map[string]Login   `mapstructure:"logins"`
	Hosts   map[string]Host    `mapstructure:"hosts"`
	Proxies ProxiesConfig      `mapstructure:"proxies"`
	Control ControlMultiConfig `mapstructure:"control"`
}

// Login holds SSH/VPN credential information.
type Login struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

// Host defines a remote endpoint to connect to.
type Host struct {
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
	Login   string `mapstructure:"login"`
	Backend string `mapstructure:"backend"`
}

// Proxy defines a single proxy endpoint configuration.
type Proxy struct {
	Port            int      `mapstructure:"port"`
	Default         string   `mapstructure:"default"`
	Allowed         []string `mapstructure:"allowed"`
	Autostart       bool     `mapstructure:"autostart"`
	AllowedControls []string `mapstructure:"allowed_controls"`
}

// ProxiesConfig holds all proxies and the global bind setting.
type ProxiesConfig struct {
	Bind    string           `mapstructure:"bind"`
	Proxies map[string]Proxy `mapstructure:",remain"`
}

// ControlLogin holds a username/token pair for controlling access to proxies.
type ControlLogin struct {
	Token string `mapstructure:"token"`
}

// ControlConfig defines how geistctl communicates with the daemon.
type ControlConfig struct {
	Mode   string                  `mapstructure:"mode"`   // "unix" or "tcp"
	Socket string                  `mapstructure:"socket"` // path to unix socket
	Listen string                  `mapstructure:"listen"` // only used if mode == "tcp"
	Auth   AuthSettings            `mapstructure:"auth"`
	Logins map[string]ControlLogin `mapstructure:"logins"`
}

// ControlInstance describes a single control interface (e.g. unix socket or TCP listener).
type ControlInstance struct {
	Name    string       `mapstructure:"name"`    // instance identifier
	Enabled bool         `mapstructure:"enabled"` // whether this instance is active
	Mode    string       `mapstructure:"mode"`    // "unix" or "tcp"
	Listen  string       `mapstructure:"listen"`  // address or socket path
	Auth    AuthSettings `mapstructure:"auth"`    // authentication settings
}

// ControlMultiConfig supports multiple control instances with distinct settings.
type ControlMultiConfig struct {
	Logins    map[string]ControlLogin `mapstructure:"logins"`    // known control users and tokens
	Instances []ControlInstance       `mapstructure:"instances"` // enabled control endpoints
}

// AuthSettings allows optional authentication for remote control.
type AuthSettings struct {
	Enabled bool   `mapstructure:"enabled"`
	Token   string `mapstructure:"token"`
}

// LoadConfig loads the portgeist configuration from disk using Viper.
// It searches for a file named config.yaml in the current working directory
// or common fallback paths, and unmarshals the content into a typed struct.
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add high-priority config path: ~/.portgeist
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".portgeist"))
	}

	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/portgeist")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return &cfg, nil
}
