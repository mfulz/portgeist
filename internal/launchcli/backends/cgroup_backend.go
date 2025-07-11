// Package launchbackends implements a cgroup-based launcher that
// starts a background proxy daemon (e.g. redsocks) via binaryBackend
// and wraps the actual user command inside a systemd-run cgroup slice.
package launchbackends

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/mfulz/portgeist/interfaces/ilauncher"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/spf13/cobra"
)

type cgroupBackend struct{}

func init() {
	ilauncher.RegisterBackend(&cgroupBackend{})
}

func (c *cgroupBackend) Method() string {
	return "cgroup"
}

// RegisterCliCmd registers a launcher using the cgroup backend as a CLI subcommand.
func (c *cgroupBackend) RegisterCliCmd(parent *cobra.Command, name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "Launch a command in a systemd-run cgroup with background proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Execute(name, cfg, host, port, ctx, args)
		},
	}
	parent.AddCommand(cmd)
	return cmd
}

// GetCmd returns the systemd-run command wrapping the actual user process inside a unique slice.
func (c *cgroupBackend) GetCmd(name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context, args []string) (*exec.Cmd, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing target command")
	}

	// debug
	logging.Log.Debugf("GetCmd got args: %v\n", args)

	// Generate slice name
	shortID := uuid.New().String()[:8]
	sliceName := fmt.Sprintf("portgeist-%s.slice", shortID)
	unitName := "pg-run-" + shortID

	// Build systemd-run wrapper
	runArgs := []string{
		"--slice=" + sliceName,
		"--unit=" + unitName,
		"--quiet",
		"-t",
		"--wait",
	}

	// Add full environment from host
	for _, env := range os.Environ() {
		runArgs = append(runArgs, "--setenv="+env)
	}

	runArgs = append(runArgs, args...)

	runCmd := exec.Command("systemd-run", runArgs...)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	runCmd.Stdin = os.Stdin
	runCmd.Env = os.Environ()

	// debug
	logging.Log.Debugf("runCmd: %v\n", runCmd)

	return runCmd, nil
}

// Execute launches the proxy daemon via binaryBackend and then runs the actual command in a systemd slice.
func (c *cgroupBackend) Execute(name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no launch command given")
	}

	// Start redsocks/helper
	helper, err := ilauncher.GetBackend("binary")
	if err != nil {
		return fmt.Errorf("missing binary backend for cgroup")
	}

	helperCmd, err := helper.GetCmd("proxy-helper-"+name, cfg, host, port, ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare proxy helper: %w", err)
	}

	if err := helperCmd.Start(); err != nil {
		return fmt.Errorf("failed to launch helper: %w", err)
	}

	// Give it a moment to spin up
	time.Sleep(500 * time.Millisecond)

	cmd, err := c.GetCmd(name, cfg, host, port, ctx, args)
	if err != nil {
		return err
	}

	err = cmd.Run()

	// Cleanup helper
	if helperCmd.Process != nil {
		err = helperCmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to kill helper: %w", err)
		}
	}

	return err
}

func (c *cgroupBackend) ExecuteBak(name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no launch command given")
	}

	// Start redsocks/helper daemon
	helper, err := ilauncher.GetBackend("binary")
	if err != nil {
		return fmt.Errorf("missing binary backend for cgroup")
	}

	go func() {
		_ = helper.Execute("proxy-helper-"+name, cfg, host, port, ctx, nil)
	}()

	// Give time to start up
	time.Sleep(500 * time.Millisecond)

	cmd, err := c.GetCmd(name, cfg, host, port, ctx, args)
	if err != nil {
		return err
	}
	return cmd.Run()
}
