// Package proxy provides logic to manage and launch proxy connections
// defined in the configuration. It handles autostart, fallback resolution,
// and proxy lifecycle management.
package proxy

import (
	"fmt"
	"log"

	"github.com/mfulz/portgeist/interfaces"
	"github.com/mfulz/portgeist/internal/config"
	"github.com/mfulz/portgeist/protocol"
)

// activeHostByProxy keeps track of the currently active host used by a proxy.
var activeHostByProxy = make(map[string]string)

// activeProxies stores the backend-level live instances by proxy name.
var activeProxies = make(map[string]interfaces.RunningInstance)

// StopAll cleanly stops all active proxies using tracked instances.
func StopAll() {
	for name, inst := range activeProxies {
		log.Printf("[proxy] Shutting down '%s'...", name)
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
func StartAutostartProxies(cfg *config.Config) error {
	for name, proxy := range cfg.Proxies.Proxies {
		if proxy.Autostart {
			log.Printf("[proxy] Autostart enabled for '%s'", name)
			err := StartProxy(name, proxy, cfg)
			if err != nil {
				log.Printf("[proxy] Failed to start '%s': %v", name, err)
			}
		}
	}
	return nil
}

// StartProxy attempts to start a proxy via its defined backend,
// using resolved backend config and storing active instance.
func StartProxy(name string, p config.Proxy, cfg *config.Config) error {
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

	// Resolve configuration override
	var resolvedConfig map[string]any
	globalCfg := cfg.Backends[backendName]
	hostCfgOverride := hostCfg.Config
	if globalCfg != nil || hostCfgOverride != nil {
		resolvedConfig = mergeConfig(globalCfg, hostCfgOverride)
	}

	if err := backend.Configure(name, resolvedConfig); err != nil {
		return fmt.Errorf("backend configuration failed: %w", err)
	}

	activeHostByProxy[name] = p.Default

	err = backend.Start(name, p, cfg)
	if err != nil {
		return err
	}

	// Optionally track the running instance
	if r, ok := backend.(interfaces.InstanceReportingBackend); ok {
		activeProxies[name] = r.GetInstance(name)
	}

	return nil
}

// StopProxy stops a running proxy by name and clears tracked state.
func StopProxy(name string, proxyCfg config.Proxy, cfg *config.Config) error {
	hostCfg, ok := cfg.Hosts[proxyCfg.Default]
	if !ok {
		return fmt.Errorf("host '%s' not found", proxyCfg.Default)
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
	delete(activeProxies, name)

	return backend.Stop(name)
}

// GetProxyStatus returns runtime information about a proxy.
func GetProxyStatus(name string, proxyCfg config.Proxy, cfg *config.Config) (*protocol.StatusResponse, error) {
	hostCfg, ok := cfg.Hosts[proxyCfg.Default]
	if !ok {
		return nil, fmt.Errorf("host '%s' not found", proxyCfg.Default)
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

// GetProxyInfo returns static and dynamic proxy metadata.
func GetProxyInfo(name string, p config.Proxy, cfg *config.Config) (*protocol.InfoResponse, error) {
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
