// Package cmd defines CLI commands for geistctl, including proxy management.
package cmd

import (
	"fmt"

	"github.com/mfulz/portgeist/internal/control"
	"github.com/mfulz/portgeist/protocol"
	"github.com/spf13/cobra"
)

var (
	proxyName string
	hostName  string
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
	Short: "Show detailed info about a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" {
			fmt.Println("Please provide a proxy name with -p")
			return
		}
		info, err := control.GetProxyInfoWithAuth(proxyName)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Proxy: %s\nBackend: %s\nPort: %d\nDefault Host: %s\nRunning: %v (PID %d)\nAutostart: %v\nAllowed Hosts: %v\nAllowed Users: %v\n",
			info.Name, info.Backend, info.Port, info.Default, info.Running, info.PID, info.Autostart, info.Allowed, info.AllowedUsers)
	},
}

var proxyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all known proxies",
	Run: func(cmd *cobra.Command, args []string) {
		proxies, err := control.ListProxiesWithAuth()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		for _, p := range proxies {
			fmt.Println("-", p)
		}
	},
}

var proxyStartCmdOld = &cobra.Command{
	Use:   "start",
	Short: "Start a proxy by name",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" {
			fmt.Println("Please provide a proxy name with -p")
			return
		}
		_, err := control.SendCommandWithAuth(fmt.Sprintf("proxy start %s", proxyName))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Requested start of proxy: %s\n", proxyName)
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
		_, err := control.SendCommandWithAuth(fmt.Sprintf("proxy stop %s", proxyName))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
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
		status, err := control.GetProxyStatusWithAuth(proxyName)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Proxy: %s\nBackend: %s\nRunning: %v\nPID: %d\n",
			status.Name, status.Backend, status.Running, status.PID)
	},
}

var proxySetActiveCmd = &cobra.Command{
	Use:   "setactive",
	Short: "Set the active host for a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if proxyName == "" || hostName == "" {
			fmt.Println("Please provide proxy name (-p) and host (-h)")
			return
		}
		_, err := control.SendCommandWithAuth(fmt.Sprintf("proxy setactive %s %s", proxyName, hostName))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Active host for '%s' set to '%s'\n", proxyName, hostName)
	},
}

func init() {
	proxyInfoCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStartCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStopCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStatusCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxySetActiveCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxySetActiveCmd.Flags().StringVarP(&hostName, "host", "n", "", "Host name")

	ProxyCmd.AddCommand(proxyInfoCmd)
	ProxyCmd.AddCommand(proxyListCmd)
	ProxyCmd.AddCommand(proxyStartCmd)
	ProxyCmd.AddCommand(proxyStopCmd)
	ProxyCmd.AddCommand(proxyStatusCmd)
	ProxyCmd.AddCommand(proxySetActiveCmd)
}
