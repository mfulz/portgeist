// Package configcli handles loading and managing local geistctl configuration.
// This includes user tokens and known daemon connection targets.
package configcli

import (
	"fmt"

	"github.com/mfulz/portgeist/internal/configloader"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/spf13/viper"
)

// UserConfig represents authentication info for a specific logical user.
type UserConfig struct {
	Username string `mapstructure:"username"`
	Token    string `mapstructure:"token"`
}

// DaemonConfig represents one connection target (unix socket or TCP).
type DaemonConfig struct {
	Socket string `mapstructure:"socket,omitempty"`
	TCP    string `mapstructure:"tcp,omitempty"`
}

// Config holds the entire client-side geistctl configuration.
type Config struct {
	Users   map[string]UserConfig   `mapstructure:"users"`
	Daemons map[string]DaemonConfig `mapstructure:"daemons"`
	Logger  logging.Config          `mapstructure:"log"`
}

// LoadConfig loads the portgeist configuration from disk using Viper.
// It searches for a file named config.yaml in the current working directory
// or common fallback paths, and unmarshals the content into a typed struct.
func LoadConfig() error {
	path, err := configloader.ResolveConfigPath("geistctl", "geistctl.yaml")
	if err != nil {
		return err
	}

	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("config unmarshal failed: %w", err)
	}

	configCfg, ok := configloader.TryGetConfig[*logging.Config]()
	if ok {
		*configCfg = cfg.Logger
	} else {
		configloader.RegisterConfig(&cfg.Logger)
	}
	err = logging.Init()
	if err != nil {
		return fmt.Errorf("[geistd] Failed to init logger: %v", err)
	}

	configloader.RegisterConfig(&cfg)
	return nil
}
