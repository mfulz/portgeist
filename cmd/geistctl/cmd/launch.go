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

var (
	proxyIp   string
	proxyPort int
)

func init() {
	LaunchCmd.PersistentFlags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	LaunchCmd.PersistentFlags().StringVarP(&daemonName, "daemon", "d", "", "Daemon name from ctl_config")
	LaunchCmd.PersistentFlags().StringVarP(&controlUser, "user", "u", "admin", "Control user to authenticate as")
	LaunchCmd.PersistentFlags().StringVar(&overrideAddr, "addr", "", "Direct override address for daemon (unix socket or host:port)")
	LaunchCmd.PersistentFlags().StringVar(&overrideToken, "token", "", "Auth token for manually specified daemon")

	LaunchCmd.PersistentFlags().StringVarP(&proxyIp, "ip", "I", "", "Override proxy host if no proxy is specified")
	LaunchCmd.PersistentFlags().IntVarP(&proxyPort, "port", "P", 0, "Override proxy port if no proxy is specified")

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

		var resolvedHost string
		var resolvedPort int

		if proxyName != "" {
			ctlcfg := configloader.MustGetConfig[*configcli.Config]()
			resolve, err := controlcli.ResolveProxy(proxyName, ctlcfg, daemonName, overrideAddr, overrideToken, controlUser)
			if err != nil {
				return fmt.Errorf("ipc error during resolve: %w", err)
			}
			logging.Log.Infof("Resolved: %v", resolve)

			if proxyIp != "" {
				resolvedHost = proxyIp
			} else {
				resolvedHost = resolve.Host
			}
			if proxyPort != 0 {
				resolvedPort = proxyPort
			} else {
				resolvedPort = resolve.Port
			}
		} else {
			if proxyPort == 0 {
				return fmt.Errorf("either --proxy or --port must be specified")
			}
			if proxyHost == "" {
				proxyHost = "127.0.0.1"
			}
			resolvedHost = proxyHost
			resolvedPort = proxyPort
		}

		ctx := ilauncher.Context{
			ProxyName:     proxyName,
			DaemonName:    daemonName,
			ControlUser:   controlUser,
			OverrideAddr:  overrideAddr,
			OverrideToken: overrideToken,
		}

		var subcmd *cobra.Command
		for aname, aLauncherCfg := range cfg.Launchers {
			backend, err := ilauncher.GetBackend(aLauncherCfg.Method)
			if err != nil {
				logging.Log.Errorf("unknown backend for launcher '%s': %v", aname, err)
				continue
			}
			if aname == name {
				subcmd = backend.RegisterCliCmd(LaunchCmd, name, *launcherCfg, resolvedHost, resolvedPort, ctx)
			}
		}

		if subcmd != nil {
			return subcmd.Execute()
		}
		return nil
	}
}

// startsWithDash checks whether a string is a flag (e.g., starts with "-")
func startsWithDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}
