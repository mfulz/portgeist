package controlcli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type ClientConfig struct {
	User   string `mapstructure:"user"`
	Token  string `mapstructure:"token"`
	Socket string `mapstructure:"socket"`
}

var cfg *ClientConfig

func LoadClientConfig() (*ClientConfig, error) {
	if cfg != nil {
		return cfg, nil
	}

	home, _ := os.UserHomeDir()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Join(home, ".portgeistctl"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var c ClientConfig
	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg = &c
	return cfg, nil
}
