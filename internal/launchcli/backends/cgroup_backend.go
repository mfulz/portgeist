// Package launchbackends implements a cgroup-based launcher that
// starts a background proxy daemon (e.g. redsocks) via binaryBackend
// and wraps the actual user command inside a systemd-run cgroup slice.
package launchbackends

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mfulz/portgeist/interfaces/ilauncher"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/spf13/cobra"
)

var (
	autoroute string
	sliceName string
	unitName  string
)

type cgroupBackend struct{}

func setupRouting(port int, slice string) error {
	if autoroute == "" {
		return nil
	}

	// Wait until cgroup.procs contains valid PID
	cgroupPath := "/sys/fs/cgroup/portgeist.slice/" + sliceName + "/" + unitName + ".service/cgroup.procs"
	var pid int
	deadline := time.Now().Add(20 * time.Second)

	for {
		logging.Log.Debugln("waiting for slice")
		data, err := os.ReadFile(cgroupPath)
		if err == nil && len(data) > 0 {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if pidStr := strings.TrimSpace(line); pidStr != "" {
					if parsed, err := strconv.Atoi(pidStr); err == nil {
						pid = parsed
						goto resolve
					}
				}
			}
		}

		if time.Now().After(deadline) {
			err := fmt.Errorf("timeout waiting for cgroup PID in %s", cgroupPath)
			logging.Log.Errorf("deadline reached: %v", err)
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}

resolve:
	hex := fmt.Sprintf("0x%x", pid)

	var cmd *exec.Cmd
	switch autoroute {
	case "nftables":
		cmd = exec.Command("sudo", "nft", "add", "rule", "ip", "nat", "output", "meta", "cgroup", hex, "ip", "protocol", "tcp", "redirect", "to", fmt.Sprintf("%d", port))
	case "iptables":
		cmd = exec.Command("iptables", "-t", "nat", "-A", "OUTPUT", "-m", "cgroup", "--cgroup", hex, "-p", "tcp", "-j", "REDIRECT", "--to-ports", fmt.Sprintf("%d", port))
	default:
		err := fmt.Errorf("unsupported autoroute backend: %s", autoroute)
		logging.Log.Errorf("Routing setup failed: %v", err)
		return err
	}

	logging.Log.Debugf("routing cmd: %v", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		logging.Log.Errorf("Routing setup failed (%s): %v", autoroute, err)
		return err
	}

	logging.Log.Debugf("Routing setup success (%s)", autoroute)
	return nil
}

func setupRoutingBak(_ string, port int, slice string) error {
	if autoroute == "" {
		return nil
	}

	// Convert slice name to hex cgroup match (simplified fallback)
	cgroupPath := "/sys/fs/cgroup/system.slice/" + slice + "/cgroup.procs"
	data, err := os.ReadFile(cgroupPath)
	if err != nil {
		return fmt.Errorf("could not read cgroup.procs: %w", err)
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid pid in cgroup.procs: %w", err)
	}

	hex := fmt.Sprintf("0x%x", pid)

	switch autoroute {
	case "nftables":
		cmd := exec.Command("nft", "add", "rule", "ip", "nat", "output", "meta", "cgroup", hex, "redirect", "to", fmt.Sprintf("%d", port))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "iptables":
		cmd := exec.Command("iptables", "-t", "nat", "-A", "OUTPUT", "-m", "cgroup", "--cgroup", hex, "-p", "tcp", "-j", "REDIRECT", "--to-ports", fmt.Sprintf("%d", port))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported autoroute backend: %s", autoroute)
	}
}

func init() {
	ilauncher.RegisterBackend(&cgroupBackend{})
}

func (c *cgroupBackend) Method() string {
	return "cgroup"
}

// RegisterCliCmd registers a launcher using the cgroup backend as a CLI subcommand.
func (c *cgroupBackend) RegisterCliCmd(parent *cobra.Command, name string, cfg ilauncher.FileConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "Launch a command in a systemd-run cgroup with background proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Execute(name, cfg, args)
		},
	}
	cmd.PersistentFlags().StringVarP(&autoroute, "autoroute", "A", "", "Optional routing backend: 'nftables' or 'iptables'")
	parent.AddCommand(cmd)
	return cmd
}

// GetCmd returns the systemd-run command wrapping the actual user process inside a unique slice.
func (c *cgroupBackend) GetCmd(name string, cfg ilauncher.FileConfig, args []string) (*exec.Cmd, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing target command")
	}

	// debug
	logging.Log.Debugf("GetCmd got args: %v\n", args)

	// Generate slice name
	shortID := uuid.New().String()[:8]
	sliceName = fmt.Sprintf("portgeist-%s.slice", shortID)
	unitName = "pg-run-" + shortID

	// Build systemd-run wrapper
	runArgs := []string{
		"--slice=" + sliceName,
		"--unit=" + unitName,
		"--property=CPUAccounting=yes",
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
func (c *cgroupBackend) Execute(name string, cfg ilauncher.FileConfig, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no launch command given")
	}

	// Start redsocks/helper
	helper, err := ilauncher.GetBackend("binary")
	if err != nil {
		return fmt.Errorf("missing binary backend for cgroup")
	}

	helperCmd, err := helper.GetCmd("proxy-helper-"+name, cfg, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare proxy helper: %w", err)
	}

	if err := helperCmd.Start(); err != nil {
		return fmt.Errorf("failed to launch helper: %w", err)
	}

	// Give it a moment to spin up
	time.Sleep(500 * time.Millisecond)

	cmd, err := c.GetCmd(name, cfg, args)
	if err != nil {
		return err
	}

	// Setup nftables or iptables route if requested
	go func() {
		setupRouting(ilauncher.Ctx.ProxyPort, sliceName)
	}()

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
