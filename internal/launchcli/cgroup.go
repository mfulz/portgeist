// Package launchcli provides CLI commands to launch processes with network isolation using Linux cgroups
// and redirection via redsocks through iptables or nftables.
//
// Example usage:
//
//	geistctl launch cgroup --iptables --redsocks-port=12345 --proxy=pp -- curl http://example.com
//
// This command creates a Linux cgroup, sets up traffic redirection through redsocks, and launches the
// specified process inside the cgroup. On exit, all rules and services are cleaned up automatically.
package launchcli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// CgroupLaunchCmd defines the 'geistctl launch cgroup' subcommand.
var CgroupLaunchCmd = &cobra.Command{
	Use:   "cgroup",
	Short: "Launch a process within an isolated cgroup with proxy redirection via redsocks",
	Long: `Launch a command inside a Linux cgroup with traffic redirected through redsocks using iptables or nftables.
By default, no redirection is applied unless --iptables or --nftables is explicitly specified.`,
	RunE: runCgroupLaunch,
}

func init() {
	CgroupLaunchCmd.Flags().Bool("iptables", false, "Use iptables for traffic redirection")
	CgroupLaunchCmd.Flags().Bool("nftables", false, "Use nftables for traffic redirection (not implemented)")
	CgroupLaunchCmd.Flags().Int("redsocks-port", 0, "Local port for redsocks to listen on")
	CgroupLaunchCmd.Flags().String("proxy", "", "Proxy alias to resolve (e.g., 'pp')")
	CgroupLaunchCmd.Flags().SetInterspersed(false)
}

// runCgroupLaunch executes the launch logic based on user-provided flags.
func runCgroupLaunch(cmd *cobra.Command, args []string) error {
	return nil
	// if len(args) == 0 {
	// 	return fmt.Errorf("no command specified to launch")
	// }
	//
	// useIptables, _ := cmd.Flags().GetBool("iptables")
	// useNftables, _ := cmd.Flags().GetBool("nftables")
	// redsocksPort, _ := cmd.Flags().GetInt("redsocks-port")
	// proxyAlias, _ := cmd.Flags().GetString("proxy")
	//
	// if useIptables && useNftables {
	// 	return fmt.Errorf("cannot use both --iptables and --nftables")
	// }
	//
	// backend := ""
	// switch {
	// case useIptables:
	// 	backend = "iptables"
	// case useNftables:
	// 	backend = "nftables"
	// default:
	// 	backend = "none"
	// }
	//
	// log := logging.Log
	//
	// // Resolve proxy
	// if proxyAlias == "" {
	// 	return fmt.Errorf("--proxy is required to resolve proxy address")
	// }
	// proxyCfg, err := configloader.ResolveProxy(proxyAlias)
	// if err != nil {
	// 	return fmt.Errorf("failed to resolve proxy '%s': %w", proxyAlias, err)
	// }
	// targetHostPort := proxyCfg.HostPort()
	//
	// // Determine redsocks port
	// if redsocksPort == 0 {
	// 	redsocksPort = 10000 + rand.Intn(5000)
	// }
	// log.Infof("Using redsocks port: %d", redsocksPort)
	//
	// // Create unique cgroup name
	// cgroupName := fmt.Sprintf("portgeist_%d", time.Now().UnixNano())
	// log.Infof("Creating cgroup: %s", cgroupName)
	//
	// // Create Cgroup
	// if err := createCgroup(cgroupName); err != nil {
	// 	return fmt.Errorf("failed to create cgroup: %w", err)
	// }
	//
	// // Setup redsocks
	// redsocksConfPath, err := generateRedsocksConf(redsocksPort, targetHostPort)
	// if err != nil {
	// 	return fmt.Errorf("failed to generate redsocks config: %w", err)
	// }
	// defer os.Remove(redsocksConfPath)
	// defer stopRedsocks()
	//
	// if err := startRedsocks(redsocksConfPath); err != nil {
	// 	return fmt.Errorf("failed to start redsocks: %w", err)
	// }
	//
	// // Setup backend
	// switch backend {
	// case "iptables":
	// 	if err := setupIptables(cgroupName, redsocksPort); err != nil {
	// 		return fmt.Errorf("iptables setup failed: %w", err)
	// 	}
	// 	defer cleanupIptables(cgroupName)
	// case "nftables":
	// 	return fmt.Errorf("nftables support is not implemented yet")
	// case "none":
	// 	log.Warn("No traffic redirection selected. Only cgroup isolation will be applied.")
	// }
	//
	// // Launch target command
	// return launchInCgroup(cgroupName, args)
}

// createCgroup creates a new Linux cgroup with the given name.
func createCgroup(name string) error {
	cgroupPath := filepath.Join("/sys/fs/cgroup", name)
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return err
	}
	return nil
}

// generateRedsocksConf generates a temporary redsocks configuration file for the given proxy target.
func generateRedsocksConf(port int, target string) (string, error) {
	conf := fmt.Sprintf(`
base {
 log_debug = off;
 log_info = on;
 daemon = on;
 redirector = iptables;
}
redsocks {
 local_ip = 127.0.0.1;
 local_port = %d;
 ip = %s;
 port = %s;
 type = socks5;
}`, port, strings.Split(target, ":")[0], strings.Split(target, ":")[1])

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("redsocks_%d.conf", port))
	err := os.WriteFile(tmp, []byte(conf), 0644)
	return tmp, err
}

// startRedsocks starts the redsocks service using the given configuration path.
func startRedsocks(confPath string) error {
	cmd := exec.Command("redsocks", "-c", confPath)
	return cmd.Start()
}

// stopRedsocks attempts to stop all running redsocks processes.
func stopRedsocks() {
	_ = exec.Command("pkill", "redsocks").Run()
}

// setupIptables installs iptables rules to redirect traffic from the given cgroup to the specified port.
func setupIptables(cgroupName string, port int) error {
	mark := 0x1A2B
	rules := [][]string{
		{"-t", "mangle", "-A", "OUTPUT", "-m", "cgroup", "--path", cgroupName, "-j", "MARK", "--set-mark", fmt.Sprintf("%d", mark)},
		{"-t", "nat", "-A", "OUTPUT", "-m", "mark", "--mark", fmt.Sprintf("%d", mark), "-j", "REDIRECT", "--to-ports", fmt.Sprintf("%d", port)},
	}
	for _, rule := range rules {
		if err := exec.Command("iptables", rule...).Run(); err != nil {
			return fmt.Errorf("iptables rule failed: %v", rule)
		}
	}
	return nil
}

// cleanupIptables removes previously installed iptables rules for the given cgroup.
func cleanupIptables(cgroupName string) {
	_ = exec.Command("iptables", "-t", "mangle", "-D", "OUTPUT", "-m", "cgroup", "--path", cgroupName, "-j", "MARK", "--set-mark", "6699").Run()
	_ = exec.Command("iptables", "-t", "nat", "-D", "OUTPUT", "-m", "mark", "--mark", "6699", "-j", "REDIRECT", "--to-ports", "12345").Run()
}

// launchInCgroup starts the given command inside the specified cgroup.
func launchInCgroup(cgroupName string, args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cgroupProcs := filepath.Join("/sys/fs/cgroup", cgroupName, "cgroup.procs")
	pid := fmt.Sprintf("%d", os.Getpid())
	if err := os.WriteFile(cgroupProcs, []byte(pid), 0644); err != nil {
		return fmt.Errorf("failed to assign to cgroup: %w", err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
