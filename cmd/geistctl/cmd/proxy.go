// Package cmd provides CLI commands for managing and inspecting proxies
// through the geistd control interface. It supports multi-daemon auth,
// dynamic routing and full introspection of proxy configuration.
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/mfulz/portgeist/internal/controlcli"
	"github.com/mfulz/portgeist/protocol"
	"github.com/spf13/cobra"
)

var (
	proxyName   string
	proxyHost   string
	daemonName  string
	controlUser string
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
		execWithAuth(protocol.CmdProxyStart, protocol.StartRequest{Name: proxyName}, "Requested start of proxy: %s\n")
	},
}

// proxyStopCmd stops a proxy by name.
var proxyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a proxy by name",
	Run: func(cmd *cobra.Command, args []string) {
		execWithAuth(protocol.CmdProxyStop, protocol.StopRequest{Name: proxyName}, "Requested stop of proxy: %s\n")
	},
}

// proxyStatusCmd shows status info about a proxy.
var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		resp := execWithAuth(protocol.CmdProxyStatus, protocol.StatusRequest{Name: proxyName}, "")
		if resp == nil || resp.Status != "ok" {
			return
		}
		var status protocol.StatusResponse
		bytes, _ := json.Marshal(resp.Data)
		_ = json.Unmarshal(bytes, &status)

		fmt.Printf("Proxy: %s\nBackend: %s\nRunning: %v\nPID: %d\nActive Host: %s\n",
			status.Name, status.Backend, status.Running, status.PID, status.ActiveHost)
	},
}

// proxyInfoCmd provides detailed runtime and config info for a proxy.
var proxyInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show combined config and runtime info of a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		resp := execWithAuth(protocol.CmdProxyInfo, protocol.InfoRequest{Name: proxyName}, "")
		if resp == nil || resp.Status != "ok" {
			return
		}
		var info protocol.InfoResponse
		b, _ := json.Marshal(resp.Data)
		_ = json.Unmarshal(b, &info)

		fmt.Printf("Name:         %s\nBackend:      %s\nRunning:      %v\nPID:          %d\nHost:         %s:%d\nLogin:        %s\nAllowed:      %v\nActive Host:  %s\n",
			info.Name, info.Backend, info.Running, info.PID,
			info.Host, info.Port, info.Login, info.Allowed, info.ActiveHost)
	},
}

// proxyListCmd lists all proxies visible to the current user.
var proxyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available proxies for the current user",
	Run: func(cmd *cobra.Command, args []string) {
		resp := execWithAuth(protocol.CmdProxyList, nil, "")
		if resp == nil || resp.Status != "ok" {
			return
		}
		var names []string
		data, _ := json.Marshal(resp.Data)
		_ = json.Unmarshal(data, &names)

		if len(names) == 0 {
			fmt.Println("No proxies available.")
			return
		}

		fmt.Println("Available proxies:")
		for _, name := range names {
			fmt.Printf(" - %s\n", name)
		}
	},
}

// proxySetActiveCmd sets the active host for a given proxy.
var proxySetActiveCmd = &cobra.Command{
	Use:   "setactive",
	Short: "Set active host for a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" || proxyHost == "" {
			fmt.Println("Please provide -p <proxy> and -o <host>")
			return
		}

		resp := execWithAuth(protocol.CmdProxySetActive, protocol.SetActiveRequest{
			Name: proxyName,
			Host: proxyHost,
		}, "")
		if resp != nil && resp.Status == "ok" {
			fmt.Printf("Active host for proxy '%s' set to '%s'\n", proxyName, proxyHost)
		}
	},
}

// execWithAuth is a helper to send an authenticated request to the selected daemon.
func execWithAuth(cmdType string, payload interface{}, successMsg string) *protocol.Response {
	cfg, err := controlcli.LoadCTLConfig()
	if err != nil {
		fmt.Printf("Error loading ctl config: %v\n", err)
		return nil
	}
	if daemonName == "" {
		daemonName = controlcli.GuessDefaultDaemon(cfg)
	}
	resp, err := controlcli.SendCommandWithAuth(cfg, daemonName, controlUser, cmdType, payload)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	if resp.Status != "ok" {
		fmt.Printf("Error: %s\n", resp.Error)
		return resp
	}
	if successMsg != "" {
		fmt.Printf(successMsg, proxyName)
	}
	return resp
}

func init() {
	// persistent options
	ProxyCmd.PersistentFlags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	ProxyCmd.PersistentFlags().StringVarP(&proxyHost, "host", "o", "", "Host name to use for setactive")
	ProxyCmd.PersistentFlags().StringVarP(&daemonName, "daemon", "d", "", "Daemon name from ctl_config")
	ProxyCmd.PersistentFlags().StringVarP(&controlUser, "user", "u", "admin", "Control user to authenticate as")

	// attach commands
	ProxyCmd.AddCommand(proxyStartCmd)
	ProxyCmd.AddCommand(proxyStopCmd)
	ProxyCmd.AddCommand(proxyStatusCmd)
	ProxyCmd.AddCommand(proxyInfoCmd)
	ProxyCmd.AddCommand(proxyListCmd)
	ProxyCmd.AddCommand(proxySetActiveCmd)
}
