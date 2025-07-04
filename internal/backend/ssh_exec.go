// Package backend provides concrete backend implementations for proxy launching.
// This file implements the SSH backend using exec.Command with sshpass.
// It manages active SSH tunnel processes using Go-controlled lifecycle.
package backend

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"syscall"

	"github.com/mfulz/portgeist/interfaces"
	"github.com/mfulz/portgeist/internal/config"
)

type sshExecBackend struct {
	mu    sync.Mutex
	procs map[string]*exec.Cmd
}

func init() {
	interfaces.RegisterBackend("ssh_exec", &sshExecBackend{
		procs: make(map[string]*exec.Cmd),
	})
}

// Start launches an SSH-based SOCKS proxy using sshpass and the given config.
// It starts the process in foreground and manages its lifecycle.
func (s *sshExecBackend) Start(name string, p config.Proxy, cfg *config.Config) error {
	hostName := p.Default
	host, ok := cfg.Hosts[hostName]
	if !ok {
		return fmt.Errorf("default host '%s' not found for proxy '%s'", hostName, name)
	}

	login, ok := cfg.Logins[host.Login]
	if !ok {
		return fmt.Errorf("login '%s' not found for host '%s'", host.Login, hostName)
	}

	bind := cfg.Proxies.Bind
	localAddr := fmt.Sprintf("%s:%d", bind, p.Port)
	remoteAddr := fmt.Sprintf("%s@%s", login.User, host.Address)

	log.Printf("[ssh_exec] Launching SOCKS proxy '%s' on %s via %s", name, localAddr, remoteAddr)

	cmd := exec.Command(
		"sshpass", "-p", login.Password,
		"ssh",
		"-N",
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-oConnectTimeout=5",
		"-D", localAddr,
		remoteAddr,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ssh start failed: %w", err)
	}

	// Track and monitor the process
	go func() {
		_ = cmd.Wait()
		log.Printf("[ssh_exec] Proxy '%s' exited", name)
	}()

	s.mu.Lock()
	s.procs[name] = cmd
	s.mu.Unlock()

	log.Printf("[ssh_exec] Proxy '%s' started (PID %d)", name, cmd.Process.Pid)
	return nil
}

// Stop attempts to terminate the SSH tunnel associated with the given proxy name.
func (s *sshExecBackend) Stop(name string) error {
	s.mu.Lock()
	cmd, ok := s.procs[name]
	s.mu.Unlock()

	if !ok {
		log.Printf("[ssh_exec] No active process found for proxy '%s'", name)
		return nil
	}

	log.Printf("[ssh_exec] Stopping proxy '%s' (PID %d)", name, cmd.Process.Pid)

	// Kill the process group
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill process for '%s': %w", name, err)
	}

	s.mu.Lock()
	delete(s.procs, name)
	s.mu.Unlock()

	log.Printf("[ssh_exec] Proxy '%s' stopped successfully", name)
	return nil
}
