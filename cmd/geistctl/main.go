// Command geistctl provides CLI control over the Portgeist daemon.
// It communicates via a configured control interface (e.g. Unix socket)
// and provides commands to list, start, stop, and configure proxies.
package main

import (
	"fmt"
	"os"

	"github.com/mfulz/portgeist/cmd/geistctl/cmd"
	"github.com/mfulz/portgeist/internal/controlcli"
	"github.com/mfulz/portgeist/internal/logging"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "geistctl",
	Short: "Control interface for the Portgeist daemon",
	Long:  `geistctl allows you to inspect and manage dynamic proxy connections handled by geistd.`,
}

func main() {
	var err error
	controlcli.CtlCfg, err = controlcli.LoadCTLConfig()
	if err != nil {
		logging.Init(logging.Config{})
		logging.Log.Errorf("[geistctl] Failed to load config: %v", err)
		os.Exit(1)
	}

	err = logging.Init(controlcli.CtlCfg.Logger)
	if err != nil {
		logging.Init(logging.Config{})
		logging.Log.Errorf("[geistctl] Failed to init logger: %v", err)
		os.Exit(1)
	}
	logging.Log.Infof("[geistctl] Log Config: %v", controlcli.CtlCfg.Logger)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(cmd.ProxyCmd)
	rootCmd.AddCommand(cmd.LaunchCmd)
}
