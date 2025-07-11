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

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// if len(args) == 0 {
		// 	return fmt.Errorf("no command provided")
		// }

		// cfg, err := launchcli.LoadLauncherConfig()
		// if err != nil {
		// 	return fmt.Errorf("failed to load config: %w", err)
		// }

		// name := cfg.Default
		// if len(args) > 1 && !startsWithDash(args[0]) {
		// 	name = args[0]
		// 	args = args[1:]
		// }
		//
		// launcherCfg, ok := cfg.Launchers[name]
		// if !ok {
		//L	return fmt.Errorf("unknown launcher: %s", name)
		// }

		if ilauncher.Ctx.ProxyName != "" {
			ctlcfg := configloader.MustGetConfig[*configcli.Config]()
			resolve, err := controlcli.ResolveProxy(
				ilauncher.Ctx.ProxyName, ctlcfg,
				ilauncher.Ctx.DaemonName, ilauncher.Ctx.ControlAddr,
				ilauncher.Ctx.UserToken, ilauncher.Ctx.ControlUser,
			)
			if err != nil {
				return fmt.Errorf("ipc error during resolve: %w", err)
			}
			logging.Log.Infof("Resolved: %v", resolve)

			if ilauncher.Ctx.ProxyIP == "" {
				ilauncher.Ctx.ProxyIP = resolve.Host
			}
			if ilauncher.Ctx.ProxyPort == 0 {
				ilauncher.Ctx.ProxyPort = resolve.Port
			}
		} else {
			if ilauncher.Ctx.ProxyPort == 0 {
				return fmt.Errorf("either --proxy or --port must be specified")
			}
			if ilauncher.Ctx.ProxyIP == "" {
				ilauncher.Ctx.ProxyIP = "127.0.0.1"
			}
		}

		// if subcmd != nil {
		// 	return subcmd.Execute()
		// }
		return nil
	},
}

var (
	proxyIp   string
	proxyPort int
)

func init() {
	LaunchCmd.PersistentFlags().StringVarP(&ilauncher.Ctx.ProxyName, "proxy", "p", "", "Proxy name")
	LaunchCmd.PersistentFlags().StringVarP(&ilauncher.Ctx.DaemonName, "daemon", "d", "", "Daemon name from ctl_config")
	LaunchCmd.PersistentFlags().StringVarP(&ilauncher.Ctx.ControlUser, "user", "u", "admin", "Control user to authenticate as")
	LaunchCmd.PersistentFlags().StringVarP(&ilauncher.Ctx.ControlAddr, "addr", "a", "", "Direct override address for daemon (unix socket or host:port)")
	LaunchCmd.PersistentFlags().StringVar(&ilauncher.Ctx.UserToken, "token", "", "Auth token for manually specified daemon")
	LaunchCmd.PersistentFlags().StringVarP(&ilauncher.Ctx.ProxyIP, "ip", "I", "", "Override proxy host if no proxy is specified")
	LaunchCmd.PersistentFlags().IntVarP(&ilauncher.Ctx.ProxyPort, "port", "P", 0, "Override proxy port if no proxy is specified")

	cfg, err := launchcli.LoadLauncherConfig()
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	// var subcmd *cobra.Command
	for aname, aLauncherCfg := range cfg.Launchers {
		backend, err := ilauncher.GetBackend(aLauncherCfg.Method)
		if err != nil {
			logging.Log.Errorf("unknown backend for launcher '%s': %v", aname, err)
			continue
		}
		backend.RegisterCliCmd(LaunchCmd, aname, *aLauncherCfg)
	}
}

// startsWithDash checks whether a string is a flag (e.g., starts with "-")
func startsWithDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}
