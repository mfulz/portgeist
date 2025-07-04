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

var proxyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all known proxies",
	Run: func(cmd *cobra.Command, args []string) {
		proxies, err := control.ListProxies()
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
		err := control.SendCommand(fmt.Sprintf("proxy start %s", proxyName))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Requested start of proxy: %s\n", proxyName)
	},
}

var proxyName string

func init() {
	proxyStartCmd.Flags().StringVarP(&proxyName, "proxy", "p", "", "Proxy name")

	ProxyCmd.AddCommand(proxyListCmd)
	ProxyCmd.AddCommand(proxyStartCmd)
}
