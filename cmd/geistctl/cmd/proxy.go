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

func init() {
	ProxyCmd.AddCommand(proxyListCmd)
}
