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
	mu           sync.Mutex
	procs        map[string]*exec.Cmd
	settings     map[string]map[string]any
	stopFlags    map[string]bool
	exitCallback func(name string)
}

func init() {
	interfaces.RegisterBackend("ssh_exec", &sshExecBackend{
		procs:     make(map[string]*exec.Cmd),
		settings:  make(map[string]map[string]any),
		stopFlags: make(map[string]bool),
	})
}

// sshInstance wraps an *exec.Cmd to support graceful Stop via interface.
type sshInstance struct {
	cmd *exec.Cmd
}

// Stop sends SIGTERM to the process associated with the instance.
func (s *sshInstance) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(syscall.SIGTERM)
	}
}

// SetExitHandler registers a callback for unexpected process exits.
func (s *sshExecBackend) SetExitHandler(cb func(name string)) {
	s.exitCallback = cb
}

// Configure stores backend-specific config per proxy instance.
func (s *sshExecBackend) Configure(name string, cfg map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[name] = cfg
	return nil
}

// Start launches the SSH tunnel process for a proxy.
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

	s.mu.Lock()
	cfgMap := s.settings[name]
	if cfgMap == nil {
		cfgMap = make(map[string]any)
	}
	s.mu.Unlock()

	key := func(opt string, fallback string) string {
		if val, ok := cfgMap[opt]; ok {
			return fmt.Sprintf("%v", val)
		}
		return fallback
	}

	connectTimeout := key("connect_timeout", "5")
	sshBinary := key("ssh_binary", "ssh")
	sshpassBinary := key("sshpass_binary", "sshpass")

	log.Printf("[ssh_exec] Launching SOCKS proxy '%s' on %s via %s", name, localAddr, remoteAddr)

	cmd := exec.Command(
		sshpassBinary, "-p", login.Password,
		sshBinary,
		"-N",
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-oConnectTimeout="+connectTimeout,
		"-D", localAddr,
		remoteAddr,
	)

	if rawFlags, ok := cfgMap["additional_flags"]; ok {
		if list, ok := rawFlags.([]interface{}); ok {
			for _, v := range list {
				if str, ok := v.(string); ok {
					cmd.Args = append(cmd.Args, str)
				}
			}
		}
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ssh start failed: %w", err)
	}

	s.mu.Lock()
	s.procs[name] = cmd
	s.stopFlags[name] = false
	s.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		log.Printf("[ssh_exec] Proxy '%s' exited", name)

		s.mu.Lock()
		intentional := s.stopFlags[name]
		delete(s.procs, name)
		delete(s.stopFlags, name)
		s.mu.Unlock()

		if !intentional && s.exitCallback != nil {
			s.exitCallback(name)
		}
	}()

	log.Printf("[ssh_exec] Proxy '%s' started (PID %d)", name, cmd.Process.Pid)
	return nil
}

// Stop attempts to terminate the SSH tunnel for the given proxy.
func (s *sshExecBackend) Stop(name string) error {
	s.mu.Lock()
	s.stopFlags[name] = true
	cmd, ok := s.procs[name]
	s.mu.Unlock()

	if !ok {
		log.Printf("[ssh_exec] No active process found for proxy '%s'", name)
		return nil
	}

	log.Printf("[ssh_exec] Stopping proxy '%s' (PID %d)", name, cmd.Process.Pid)

	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill process for '%s': %w", name, err)
	}

	log.Printf("[ssh_exec] Proxy '%s' stop signal sent", name)
	return nil
}

// Status returns PID and running state of a proxy.
func (s *sshExecBackend) Status(name string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd, ok := s.procs[name]
	if !ok || cmd.Process == nil {
		return 0, false
	}
	return cmd.Process.Pid, true
}

// GetInstance returns a RunningInstance for the proxy, if active.
func (s *sshExecBackend) GetInstance(name string) interfaces.RunningInstance {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd, ok := s.procs[name]
	if !ok {
		return nil
	}
	return &sshInstance{cmd: cmd}
}
