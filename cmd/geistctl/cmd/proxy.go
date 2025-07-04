// Package cmd defines CLI commands for geistctl, including proxy management.
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/mfulz/portgeist/internal/control"
	"github.com/mfulz/portgeist/protocol"
	"github.com/spf13/cobra"
)

var (
	proxyName string
	proxyHost string
)

var ProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage and inspect proxies",
}

var proxyStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a proxy by name",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" {
			fmt.Println("Please provide a proxy name with -p")
			return
		}

		req := protocol.Request{
			Type: protocol.CmdProxyStart,
			Data: protocol.StartRequest{Name: proxyName},
		}

		resp, err := control.SendStructuredRequest(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if resp.Status != "ok" {
			fmt.Printf("Error: %s\n", resp.Error)
			return
		}

		fmt.Printf("Requested start of proxy: %s\n", proxyName)
	},
}

var proxyInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show combined config and runtime info of a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" {
			fmt.Println("Please provide a proxy name with -p")
			return
		}

		req := protocol.Request{
			Type: protocol.CmdProxyInfo,
			Data: protocol.InfoRequest{Name: proxyName},
		}

		resp, err := control.SendStructuredRequest(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if resp.Status != "ok" {
			fmt.Printf("Error: %s\n", resp.Error)
			return
		}

		var info protocol.InfoResponse
		b, _ := json.Marshal(resp.Data)
		_ = json.Unmarshal(b, &info)

		fmt.Printf("Name:     %s\nBackend:  %s\nRunning:  %v\nPID:      %d\nHost:     %s:%d\nLogin:    %s\nAllowed:  %v\n",
			info.Name, info.Backend, info.Running, info.PID,
			info.Host, info.Port, info.Login, info.Allowed)
	},
}

var proxyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a proxy by name",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" {
			fmt.Println("Please provide a proxy name with -p")
			return
		}

		req := protocol.Request{
			Type: protocol.CmdProxyStop,
			Data: protocol.StopRequest{Name: proxyName},
		}

		resp, err := control.SendStructuredRequest(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if resp.Status != "ok" {
			fmt.Printf("Error: %s\n", resp.Error)
			return
		}

		fmt.Printf("Requested stop of proxy: %s\n", proxyName)
	},
}

var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" {
			fmt.Println("Please provide a proxy name with -p")
			return
		}

		req := protocol.Request{
			Type: protocol.CmdProxyStatus,
			Data: protocol.StatusRequest{Name: proxyName},
		}

		resp, err := control.SendStructuredRequest(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if resp.Status != "ok" {
			fmt.Printf("Error: %s\n", resp.Error)
			return
		}

		var status protocol.StatusResponse
		bytes, _ := json.Marshal(resp.Data)
		_ = json.Unmarshal(bytes, &status)

		fmt.Printf("Proxy: %s\nBackend: %s\nRunning: %v\nPID: %d\n",
			status.Name, status.Backend, status.Running, status.PID)
	},
}

var proxyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available proxies for the current user",
	Run: func(cmd *cobra.Command, args []string) {
		req := protocol.Request{
			Type: protocol.CmdProxyList,
		}

		resp, err := control.SendStructuredRequest(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if resp.Status != "ok" {
			fmt.Printf("Error: %s\n", resp.Error)
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

var proxySetActiveCmd = &cobra.Command{
	Use:   "setactive",
	Short: "Set active host for a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" || proxyHost == "" {
			fmt.Println("Please provide -p <proxy> and -h <host>")
			return
		}

		req := protocol.Request{
			Type: protocol.CmdProxySetActive,
			Data: protocol.SetActiveRequest{
				Name: proxyName,
				Host: proxyHost,
			},
		}

		resp, err := control.SendStructuredRequest(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if resp.Status != "ok" {
			fmt.Printf("Error: %s\n", resp.Error)
			return
		}

		fmt.Printf("Active host for proxy '%s' set to '%s'\n", proxyName, proxyHost)
	},
}

func init() {
	proxyInfoCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStartCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStopCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStatusCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxySetActiveCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxySetActiveCmd.PersistentFlags().StringVarP(&proxyHost, "host", "o", "", "Host name to use for setactive")

	ProxyCmd.AddCommand(proxyInfoCmd)
	ProxyCmd.AddCommand(proxyListCmd)
	ProxyCmd.AddCommand(proxyStartCmd)
	ProxyCmd.AddCommand(proxyStopCmd)
	ProxyCmd.AddCommand(proxyStatusCmd)
	ProxyCmd.AddCommand(proxySetActiveCmd)
}
