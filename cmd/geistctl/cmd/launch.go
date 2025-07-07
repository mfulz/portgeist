// Package cmd provides CLI commands for the geistctl binary.
// This file defines the "launch" subcommand for starting wrapped processes.
package cmd

import (
	"os"

	"github.com/mfulz/portgeist/internal/launchcli"
	"github.com/mfulz/portgeist/internal/logging"

	"github.com/spf13/cobra"
)

// LaunchCmd launches an arbitrary command through a proxy wrapper tool.
// It loads config from ~/.portgeist/geistctl/launch.yaml and supports
// dynamic proxychains/torsocks configuration with per-launch isolation.
var LaunchCmd = &cobra.Command{
	Use:   "launch -- <command> [args...]",
	Short: "Launch a command through proxy wrapper (e.g. proxychains)",
	Long: `Wraps a user-specified command with a proxy tool like proxychains.

Example:
  geistctl launch -- curl https://ifconfig.me
  geistctl launch -- alacritty --class testenv`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logging.Log.Errorln("Missing command to execute.")
			os.Exit(1)
		}

		cfg, err := launchcli.LoadFileConfig()
		if err != nil {
			logging.Log.Errorf("Config error: %v\n", err)
			os.Exit(1)
		}

		var confPath string
		if cfg.ConfigTemplate != "" {
			confPath, err = launchcli.GenerateProxyConf(cfg.ConfigTemplate, 8889)
			if err != nil {
				logging.Log.Errorf("Failed to create proxy config: %v\n", err)
				os.Exit(1)
			}
		}

		err = launchcli.Launch(launchcli.Config{
			Method:   cfg.Method,
			Binary:   cfg.Binary,
			Command:  args,
			Env:      cfg.Env,
			ConfPath: confPath,
		})
		if err != nil {
			logging.Log.Fatalf("Launch failed: %v\n", err)
			os.Exit(1)
		}
	},
}
