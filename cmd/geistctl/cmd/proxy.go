// Package cmd defines CLI commands for geistctl, including proxy management.
package cmd

import (
	"fmt"

	"github.com/mfulz/portgeist/internal/control"
	"github.com/spf13/cobra"
)

var ProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage and inspect proxies",
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

var proxyStartCmd = &cobra.Command{
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

var proxyName string

func init() {
	proxyInfoCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStartCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStopCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")
	proxyStatusCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")

	ProxyCmd.AddCommand(proxyInfoCmd)
	ProxyCmd.AddCommand(proxyListCmd)
	ProxyCmd.AddCommand(proxyStartCmd)
	ProxyCmd.AddCommand(proxyStopCmd)
	ProxyCmd.AddCommand(proxyStatusCmd)
}
