// Package launchcli contains logic to execute external commands
// using tools like proxychains or torsocks. It supports injecting environment
// variables and dynamic configuration file paths.
package launchcli

import (
	"fmt"
	"os"
	"os/exec"
)

// Config defines a fully prepared launch environment for a proxy-wrapped execution.
type Config struct {
	Method   string            // Method name (e.g. proxychains)
	Binary   string            // Executable binary (e.g. /usr/bin/proxychains)
	Env      map[string]string // Environment variables to apply
	Command  []string          // Actual command to launch (first arg = binary)
	ConfPath string            // Optional config file path (e.g. proxychains.conf)
}

// Launch executes the configured command using the specified wrapper binary.
// It applies environment variables and injects a config file path if required.
func Launch(cfg Config) error {
	cmd := exec.Command(cfg.Binary, cfg.Command...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	if cfg.ConfPath != "" {
		switch cfg.Method {
		case "proxychains":
			env = append(env, "PROXYCHAINS_CONF_FILE="+cfg.ConfPath)
		case "torsocks":
			env = append(env, "TORSOCKS_CONF_FILE="+cfg.ConfPath)
		}
	}
	cmd.Env = env
	return cmd.Run()
}
