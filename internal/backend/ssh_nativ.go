// Package backend provides concrete backend implementations for proxy launching.
// This file implements the SSH backend using native Go SSH functionality.
package backend

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-socks5"
	"github.com/mfulz/portgeist/interfaces"
	"github.com/mfulz/portgeist/internal/configd"
	"github.com/mfulz/portgeist/internal/logging"

	"golang.org/x/crypto/ssh"
)

type sshNativeBackend struct {
	mu           sync.Mutex
	procs        map[string]ssh.Conn
	settings     map[string]map[string]any
	stopFlags    map[string]bool
	exitCallback func(name string)
}

func init() {
	interfaces.RegisterBackend("ssh_native", &sshNativeBackend{
		procs:     make(map[string]ssh.Conn),
		settings:  make(map[string]map[string]any),
		stopFlags: make(map[string]bool),
	})
}

// sshInstance wraps a net.Conn to support graceful Stop via interface.
type sshNativeInstance struct {
	conn ssh.Conn
}

// Stop sends SIGTERM to the process associated with the instance.
func (s *sshNativeInstance) Stop() {
	if s.conn != nil && s.conn.RemoteAddr() != nil {
		_ = s.conn.Close()
	}
}

// SetExitHandler registers a callback for unexpected process exits.
func (s *sshNativeBackend) SetExitHandler(cb func(name string)) {
	s.exitCallback = cb
}

// Configure stores backend-specific config per proxy instance.
func (s *sshNativeBackend) Configure(name string, cfg map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[name] = cfg
	return nil
}

// Start launches the SSH tunnel process for a proxy.
func (s *sshNativeBackend) Start(name string, p configd.Proxy, cfg *configd.Config) error {
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
	remoteAddr := host.Address
	if !strings.HasSuffix(host.Address, ":") {
		remoteAddr = fmt.Sprintf("%s:22", host.Address)
	}

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

	connectTimeoutStr := key("connect_timeout", "5")
	connectTimeout, err := strconv.Atoi(connectTimeoutStr)
	if err != nil {
		connectTimeout = 5
	}

	sshConf := &ssh.ClientConfig{
		User:            login.User,
		Auth:            []ssh.AuthMethod{ssh.Password(login.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * time.Duration(connectTimeout),
	}

	logging.Log.Infof("[ssh_native] Launching SOCKS proxy '%s' on %s via %s", name, localAddr, remoteAddr)

	conn, err := ssh.Dial("tcp", remoteAddr, sshConf)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	s.mu.Lock()
	s.procs[name] = conn
	s.stopFlags[name] = false
	s.mu.Unlock()

	go func() {
		conf := &socks5.Config{
			Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return conn.Dial(network, addr)
			},
		}

		serverSocks, err := socks5.New(conf)
		if err != nil {
			logging.Log.Errorf("[ssh_native] Socks error '%s'", err)
			conn.Close()
		}
		if err := serverSocks.ListenAndServe("tcp", localAddr); err != nil {
			logging.Log.Errorf("[ssh_native] Socks error '%s'", err)
			conn.Close()
		}

		_ = conn.Wait()
		logging.Log.Infof("[ssh_native] Proxy '%s' exited", name)

		s.mu.Lock()
		intentional := s.stopFlags[name]
		delete(s.procs, name)
		delete(s.stopFlags, name)
		s.mu.Unlock()

		if !intentional && s.exitCallback != nil {
			s.exitCallback(name)
		}
	}()

	logging.Log.Infof("[ssh_native] Proxy '%s' started", name)
	return nil
}

// Stop attempts to terminate the SSH tunnel for the given proxy.
func (s *sshNativeBackend) Stop(name string) error {
	s.mu.Lock()
	s.stopFlags[name] = true
	conn, ok := s.procs[name]
	if ok && conn.RemoteAddr() != nil {
		conn.Close()
	}
	s.mu.Unlock()

	if !ok {
		logging.Log.Infof("[ssh_native] No active process found for proxy '%s'", name)
		return nil
	}

	logging.Log.Infof("[ssh_native] Stopping proxy '%s' (PID %d)", name, conn.RemoteAddr())

	if err := conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection for '%s': %w", name, err)
	}

	logging.Log.Infof("[ssh_native] Proxy '%s' stop signal sent", name)
	return nil
}

// Status returns PID and running state of a proxy.
func (s *sshNativeBackend) Status(name string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, ok := s.procs[name]
	if !ok || conn.RemoteAddr() == nil {
		return 0, false
	}
	return int(conn.RemoteAddr().(*net.TCPAddr).Port), true
}

// GetInstance returns a RunningInstance for the proxy, if active.
func (s *sshNativeBackend) GetInstance(name string) interfaces.RunningInstance {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, ok := s.procs[name]
	if !ok || conn.RemoteAddr() == nil {
		return nil
	}
	return &sshNativeInstance{conn: conn}
}
