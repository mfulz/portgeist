// Package launchbackends contains backend implementations for configurable launchers.
// The binaryBackend provides a generic wrapper for launching processes via external binaries
// like proxychains, torsocks, redsocks etc., using config from launchers/*.yaml.
package launchbackends

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mfulz/portgeist/interfaces/ilauncher"
	"github.com/mfulz/portgeist/internal/launchcli"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/spf13/cobra"
)

type binaryBackend struct{}

func init() {
	ilauncher.RegisterBackend(&binaryBackend{})
}

// Method returns the unique identifier for this backend.
func (b *binaryBackend) Method() string {
	return "binary"
}

// RegisterCliCmd registers a CLI subcommand under the given parent Cobra command.
// This allows the backend to expose its own usage and flags if needed.
func (b *binaryBackend) RegisterCliCmd(parent *cobra.Command, name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Launch using backend '%s'", b.Method()),
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.Execute(name, cfg, host, port, ctx, args)
		},
	}
	parent.AddCommand(cmd)
	return cmd
}

// GetCmd builds an exec.Cmd that can be launched externally.
// It renders config template and replaces placeholders as needed.
func (b *binaryBackend) GetCmd(name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context, args []string) (*exec.Cmd, error) {
	var confPath string
	if cfg.ConfigTemplate != "" {
		runport := 10000 + time.Now().UnixNano()%4000
		content := strings.ReplaceAll(cfg.ConfigTemplate, "{{RUN_PORT}}", fmt.Sprintf("%d", runport))
		content = strings.ReplaceAll(content, "{{PORT}}", fmt.Sprintf("%d", port))
		content = strings.ReplaceAll(content, "{{HOST}}", host)
		confPath = filepath.Join(os.TempDir(), fmt.Sprintf("%s_%d.conf", name, time.Now().UnixNano()))
		if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write config: %w", err)
		}
		logging.Log.Debugf("temp config:\n%s", content)
	}

	// Expand args_before with {{CONF}}
	var expandedArgs []string
	for _, arg := range cfg.ArgsBefore {
		if confPath != "" {
			arg = strings.ReplaceAll(arg, "{{CONF}}", confPath)
		}
		expandedArgs = append(expandedArgs, arg)
	}
	finalArgs := append(expandedArgs, args...)

	// Convert map[string]string to []string
	envList := os.Environ()
	for k, v := range cfg.Env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(cfg.Binary, finalArgs...)
	cmd.Env = envList
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

// Execute launches the binary using internal launch infrastructure.
// This is used for isolated, non-systemd execution paths.
func (b *binaryBackend) Execute(name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context, args []string) error {
	cmd, err := b.GetCmd(name, cfg, host, port, ctx, args)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// GetInstance returns a Cobra command instance for this launcher backend.
func (b *binaryBackend) GetInstance(name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Launch using binary backend: %s", cfg.Binary),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "Missing command to execute.")
				os.Exit(1)
			}

			var confPath string
			if cfg.ConfigTemplate != "" {
				runport := 10000 + time.Now().UnixNano()%4000
				content := strings.ReplaceAll(cfg.ConfigTemplate, "{{RUN_PORT}}", fmt.Sprintf("%d", runport))
				content = strings.ReplaceAll(content, "{{PORT}}", fmt.Sprintf("%d", port))
				content = strings.ReplaceAll(content, "{{HOST}}", host)
				confPath = filepath.Join(os.TempDir(), fmt.Sprintf("%s_%d.conf", name, time.Now().UnixNano()))
				if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to write config: %v\n", err)
					os.Exit(1)
				}

				logging.Log.Debugf("temp config:\n%s", content)
			}

			// Expand args_before with {{CONF}} if present
			var expandedArgs []string
			for _, arg := range cfg.ArgsBefore {
				if confPath != "" {
					arg = strings.ReplaceAll(arg, "{{CONF}}", confPath)
				}
				expandedArgs = append(expandedArgs, arg)
			}

			fullArgs := append(expandedArgs, args...)

			err := launchcli.Launch(launchcli.Config{
				Method:   cfg.Method,
				Binary:   cfg.Binary,
				Command:  fullArgs,
				Env:      cfg.Env,
				ConfPath: confPath,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Launch failed: %v\n", err)
				os.Exit(1)
			}

			return nil
		},
	}

	return cmd, nil
}
