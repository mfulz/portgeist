// Package proxy provides logic to manage and launch proxy connections
// defined in the configuration. It handles autostart, fallback resolution,
// and proxy lifecycle management.
package proxy

import (
	"fmt"
	"sync"
	"time"

	"github.com/mfulz/portgeist/interfaces"
	"github.com/mfulz/portgeist/internal/configd"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/mfulz/portgeist/protocol"
)

var (
	// proxyTransitionMu ensures that start/stop operations are serialized to avoid race conditions.
	proxyTransitionMu sync.Mutex

	// activeHostByProxy keeps track of the currently active host used by a proxy.
	activeHostByProxy = make(map[string]string)

	// activeProxies stores the backend-level live instances by proxy name.
	activeProxies = make(map[string]interfaces.RunningInstance)

	// stateCheckMaxWait sets the maximum wait time for stopping proxy
	stateCheckMaxWait = 15 * time.Second

	// stateCheckInterval sets the iteration wait time for stopping proxy loop
	stateCheckInterval = 100 * time.Millisecond
)

// waitUntilStopped polls backend.Status until it reports not running or timeout.
func waitUntilStopped(backend interfaces.ProxyBackend, name string) {
	timeout := time.After(stateCheckMaxWait)
	tick := time.Tick(stateCheckInterval)
	for {
		select {
		case <-timeout:
			logging.Log.Warnf("[proxy] Timeout while waiting for '%s' to stop", name)
			return
		case <-tick:
			_, running := backend.Status(name)
			if !running {
				return
			}
		}
	}
}

// StopAll cleanly stops all active proxies using tracked instances.
func StopAll() {
	for name, inst := range activeProxies {
		logging.Log.Infof("[proxy] Shutting down '%s'...", name)
		inst.Stop()
	}
}

// mergeConfig merges global and host-specific backend configuration values.
func mergeConfig(global, override map[string]any) map[string]any {
	out := make(map[string]any)
	for k, v := range global {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

// StartAutostartProxies starts all proxies marked as autostart=true
// from the provided configuration.
func StartAutostartProxies(cfg *configd.Config) error {
	for name, p := range cfg.Proxies.Proxies {
		if p.Autostart {
			logging.Log.Infof("[proxy] Autostart enabled for '%s'", name)
			if err := StartProxy(name, p, cfg); err != nil {
				logging.Log.Infof("[proxy] Failed to start '%s': %v", name, err)
			}
		}
	}
	return nil
}

// StartProxy attempts to start a proxy via its defined backend,
// using resolved backend config and storing active instance.
func StartProxy(name string, p configd.Proxy, cfg *configd.Config) error {
	proxyTransitionMu.Lock()
	defer proxyTransitionMu.Unlock()

	hostCfg, ok := cfg.Hosts[p.Default]
	if !ok {
		return fmt.Errorf("host '%s' not found for proxy '%s'", p.Default, name)
	}
	backendName := hostCfg.Backend
	if backendName == "" {
		backendName = "ssh_exec"
	}
	backend, err := interfaces.GetBackend(backendName)
	if err != nil {
		return fmt.Errorf("unknown backend '%s': %w", backendName, err)
	}

	// check if already running
	_, running := backend.Status(name)
	if running {
		logging.Log.Infof("[proxy] '%s' is already running", name)
		return nil
	}

	globalCfg := cfg.Backends[backendName]
	resolved := mergeConfig(globalCfg, hostCfg.Config)

	if err := backend.Configure(name, resolved); err != nil {
		return fmt.Errorf("backend configure failed: %w", err)
	}

	// Register restart callback if supported
	if withNotify, ok := backend.(interfaces.ExitAwareBackend); ok {
		withNotify.SetExitHandler(func(deadName string) {
			logging.Log.Infof("[proxy] Detected exit of '%s', attempting restart", deadName)
			_ = StopProxy(deadName, p, cfg)
			if err := StartProxy(deadName, p, cfg); err != nil {
				logging.Log.Infof("[proxy] Restart of '%s' failed: %v", deadName, err)
			} else {
				logging.Log.Infof("[proxy] Restarted '%s' successfully", deadName)
			}
		})
	}

	activeHostByProxy[name] = p.Default

	if err := backend.Start(name, p, cfg); err != nil {
		return err
	}

	if reporting, ok := backend.(interfaces.InstanceReportingBackend); ok {
		if inst := reporting.GetInstance(name); inst != nil {
			activeProxies[name] = inst
		}
	}
	return nil
}

// StopProxy stops a running proxy by name and clears tracked state.
func StopProxy(name string, p configd.Proxy, cfg *configd.Config) error {
	proxyTransitionMu.Lock()
	defer proxyTransitionMu.Unlock()

	hostCfg, ok := cfg.Hosts[p.Default]
	if !ok {
		return fmt.Errorf("host '%s' not found", p.Default)
	}
	backendName := hostCfg.Backend
	if backendName == "" {
		backendName = "ssh_exec"
	}
	backend, err := interfaces.GetBackend(backendName)
	if err != nil {
		return err
	}

	delete(activeHostByProxy, name)

	if err := backend.Stop(name); err != nil {
		return err
	}

	waitUntilStopped(backend, name)
	return nil
}

// GetProxyStatus returns runtime information about a proxy.
func GetProxyStatus(name string, p configd.Proxy, cfg *configd.Config) (*protocol.StatusResponse, error) {
	hostCfg, ok := cfg.Hosts[p.Default]
	if !ok {
		return nil, fmt.Errorf("host '%s' not found", p.Default)
	}
	backendName := hostCfg.Backend
	if backendName == "" {
		backendName = "ssh_exec"
	}
	backend, err := interfaces.GetBackend(backendName)
	if err != nil {
		return nil, err
	}
	pid, running := backend.Status(name)
	return &protocol.StatusResponse{
		Name:       name,
		Backend:    backendName,
		Running:    running,
		PID:        pid,
		ActiveHost: activeHostByProxy[name],
	}, nil
}

// GetProxyInfo returns static and dynamic information about a proxy,
// including its host, port, backend, credentials, allowed users and active host.
func GetProxyInfo(name string, p configd.Proxy, cfg *configd.Config) (*protocol.InfoResponse, error) {
	hostCfg, ok := cfg.Hosts[p.Default]
	if !ok {
		return nil, fmt.Errorf("host not found")
	}
	backend := hostCfg.Backend
	if backend == "" {
		backend = "ssh_exec"
	}
	be, err := interfaces.GetBackend(backend)
	if err != nil {
		return nil, err
	}
	pid, running := be.Status(name)
	return &protocol.InfoResponse{
		Name:       name,
		Backend:    backend,
		Host:       hostCfg.Address,
		Port:       hostCfg.Port,
		Login:      hostCfg.Login,
		Running:    running,
		PID:        pid,
		Allowed:    p.AllowedControls,
		ActiveHost: activeHostByProxy[name],
	}, nil
}
