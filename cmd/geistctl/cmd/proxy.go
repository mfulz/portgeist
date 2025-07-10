// Package cmd provides CLI commands for managing and inspecting proxies
// through the geistd control interface. It supports multi-daemon auth,
// dynamic routing and full introspection of proxy configuration.
package cmd

import (
	"github.com/mfulz/portgeist/internal/configcli"
	"github.com/mfulz/portgeist/internal/configloader"
	"github.com/mfulz/portgeist/internal/controlcli"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/mfulz/portgeist/protocol"
	"github.com/spf13/cobra"
)

var (
	proxyName     string
	proxyHost     string
	daemonName    string
	controlUser   string
	overrideAddr  string
	overrideToken string
)

// ProxyCmd is the root command for proxy-related subcommands.
var ProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage and inspect proxies",
}

// proxyStartCmd starts a proxy by name.
var proxyStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a proxy by name",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := configloader.MustGetConfig[*configcli.Config]()
		if err := controlcli.StartProxy(proxyName, cfg, daemonName, overrideAddr, overrideToken, controlUser); err != nil {
			logging.Log.Errorf("[geistctl] error: %v", err)
		}
	},
}

// proxyStopCmd stops a proxy by name.
var proxyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a proxy by name",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := configloader.MustGetConfig[*configcli.Config]()
		if err := controlcli.StopProxy(proxyName, cfg, daemonName, overrideAddr, overrideToken, controlUser); err != nil {
			logging.Log.Errorf("[geistctl] error: %v", err)
		}
	},
}

// proxyStatusCmd shows status info about a proxy.
var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := configloader.MustGetConfig[*configcli.Config]()
		status, err := controlcli.ProxyStatus(proxyName, cfg, daemonName, overrideAddr, overrideToken, controlUser)
		if err != nil {
			logging.Log.Errorf("[geistctl] error: %v", err)
			return
		}

		logging.Log.Infof("Proxy: %s\nBackend: %s\nRunning: %v\nPID: %d\nActive Host: %s\n",
			status.Name, status.Backend, status.Running, status.PID, status.ActiveHost)
	},
}

// proxyInfoCmd provides detailed runtime and config info for a proxy.
var proxyInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show combined config and runtime info of a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := configloader.MustGetConfig[*configcli.Config]()
		info, err := controlcli.ProxyInfo(proxyName, cfg, daemonName, overrideAddr, overrideToken, controlUser)
		if err != nil {
			logging.Log.Errorf("[geistctl] error: %v", err)
			return
		}

		logging.Log.Infof("Name:         %s\nBackend:      %s\nRunning:      %v\nPID:          %d\nHost:         %s:%d\nLogin:        %s\nAllowed:      %v\nActive Host:  %s\n",
			info.Name, info.Backend, info.Running, info.PID,
			info.Host, info.Port, info.Login, info.Allowed, info.ActiveHost)
	},
}

// proxyListCmd lists all proxies visible to the current user.
var proxyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available proxies for the current user",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := configloader.MustGetConfig[*configcli.Config]()
		list, err := controlcli.ProxyList(cfg, daemonName, overrideAddr, overrideToken, controlUser)
		if err != nil {
			logging.Log.Errorf("[geistctl] error: %v", err)
			return
		}

		if len(list.Proxies) == 0 {
			logging.Log.Warnln("No proxies available.")
			return
		}

		logging.Log.Infoln("Available proxies:")
		for _, name := range list.Proxies {
			logging.Log.Infof(" - %s\n", name)
		}
	},
}

// proxySetActiveCmd sets the active host for a given proxy.
var proxySetActiveCmd = &cobra.Command{
	Use:   "setactive",
	Short: "Set active host for a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" || proxyHost == "" {
			logging.Log.Infoln("Please provide -p <proxy> and -o <host>")
			return
		}

		cfg := configloader.MustGetConfig[*configcli.Config]()
		if err := controlcli.SetActiveProxy(proxyName, cfg, daemonName, overrideAddr, overrideToken, controlUser, proxyHost); err != nil {
			logging.Log.Errorf("[geistctl] error: %v", err)
			return
		}
		logging.Log.Infof("Active host for proxy '%s' set to '%s'\n", proxyName, proxyHost)
	},
}

// execWithAuth sends a request to the configured or overridden daemon with optional authentication.
func execWithAuth(cmdType string, payload interface{}, successMsg string) *protocol.Response {
	var err error

	cfg := configloader.MustGetConfig[*configcli.Config]()

	if daemonName == "" {
		daemonName = controlcli.GuessDefaultDaemon(cfg)
	}

	var resp *protocol.Response
	if overrideAddr != "" {
		// Use override (e.g. via --addr and --token)
		resp, err = controlcli.SendDirectCommand(overrideAddr, overrideToken, controlUser, cmdType, payload)
	} else {
		resp, err = controlcli.SendCommandWithAuth(cfg, daemonName, controlUser, cmdType, payload)
	}

	if err != nil {
		logging.Log.Infof("Error: %v\n", err)
		return nil
	}
	if resp.Status != "ok" {
		logging.Log.Infof("Error: %s\n", resp.Error)
		return resp
	}
	if successMsg != "" {
		logging.Log.Infof(successMsg, proxyName)
	}
	return resp
}

func init() {
	// persistent options
	ProxyCmd.PersistentFlags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	ProxyCmd.PersistentFlags().StringVarP(&proxyHost, "host", "o", "", "Host name to use for setactive")
	ProxyCmd.PersistentFlags().StringVarP(&daemonName, "daemon", "d", "", "Daemon name from ctl_config")
	ProxyCmd.PersistentFlags().StringVarP(&controlUser, "user", "u", "admin", "Control user to authenticate as")
	ProxyCmd.PersistentFlags().StringVar(&overrideAddr, "addr", "", "Direct override address for daemon (unix socket or host:port)")
	ProxyCmd.PersistentFlags().StringVar(&overrideToken, "token", "", "Auth token for manually specified daemon")

	// attach commands
	ProxyCmd.AddCommand(proxyStartCmd)
	ProxyCmd.AddCommand(proxyStopCmd)
	ProxyCmd.AddCommand(proxyStatusCmd)
	ProxyCmd.AddCommand(proxyInfoCmd)
	ProxyCmd.AddCommand(proxyListCmd)
	ProxyCmd.AddCommand(proxySetActiveCmd)
}
