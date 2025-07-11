// Package cmd provides CLI commands for the geistctl binary.
// This file defines the "launch" subcommand for starting wrapped processes.
package cmd

import (
	"fmt"

	"github.com/mfulz/portgeist/interfaces/ilauncher"
	"github.com/mfulz/portgeist/internal/configcli"
	"github.com/mfulz/portgeist/internal/configloader"
	"github.com/mfulz/portgeist/internal/controlcli"
	"github.com/mfulz/portgeist/internal/launchcli"
	_ "github.com/mfulz/portgeist/internal/launchcli/backends"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/spf13/cobra"
)

// LaunchCmd wraps commands using a backend-defined launcher system.
var LaunchCmd = &cobra.Command{
	Use:   "launch [launcher] -- <command> [args...]",
	Short: "Launch a command using proxy wrapper or sandbox",
	Long: `Starts a command with traffic redirection through a configured launcher backend.

Examples:
  geistctl launch proxychains -- curl http://example.com
  geistctl launch custom1 -- firefox
  geistctl launch -- curl http://ipinfo.io  # uses default`,
}

func init() {
	LaunchCmd.PersistentFlags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	LaunchCmd.PersistentFlags().StringVarP(&daemonName, "daemon", "d", "", "Daemon name from ctl_config")
	LaunchCmd.PersistentFlags().StringVarP(&controlUser, "user", "u", "admin", "Control user to authenticate as")
	LaunchCmd.PersistentFlags().StringVar(&overrideAddr, "addr", "", "Direct override address for daemon (unix socket or host:port)")
	LaunchCmd.PersistentFlags().StringVar(&overrideToken, "token", "", "Auth token for manually specified daemon")

	LaunchCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("no command provided")
		}

		cfg, err := launchcli.LoadLauncherConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name := cfg.Default
		if len(args) > 1 && !startsWithDash(args[0]) {
			name = args[0]
			args = args[1:]
		}

		launcherCfg, ok := cfg.Launchers[name]
		if !ok {
			return fmt.Errorf("unknown launcher: %s", name)
		}

		ctlcfg := configloader.MustGetConfig[*configcli.Config]()
		resolve, err := controlcli.ResolveProxy(proxyName, ctlcfg, daemonName, overrideAddr, overrideToken, controlUser)
		if err != nil {
			return fmt.Errorf("ipc error during resolve: %w", err)
		}
		logging.Log.Infof("Resolved: %v", resolve)

		backend, err := ilauncher.GetBackend(launcherCfg.Method)
		if err != nil {
			return fmt.Errorf("backend not found: %s", launcherCfg.Method)
		}

		subcmd, err := backend.GetInstance(name, *launcherCfg, resolve.Host, resolve.Port)
		if err != nil {
			return fmt.Errorf("failed to instantiate launcher: %w", err)
		}

		// we pass args as if they were passed to the subcommand directly
		subcmd.SetArgs(args)
		return subcmd.Execute()
	}
}

// startsWithDash checks whether a string is a flag (e.g., starts with "-")
func startsWithDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}
