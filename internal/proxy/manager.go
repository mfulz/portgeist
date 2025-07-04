// Package proxy provides logic to manage and launch proxy connections
// defined in the configuration. It handles autostart, fallback resolution,
// and proxy lifecycle management.
package proxy

import (
	"fmt"
	"log"

	"github.com/mfulz/portgeist/interfaces"
	"github.com/mfulz/portgeist/internal/config"
)

// StartAutostartProxies starts all proxies marked as autostart=true
// from the provided configuration. It resolves the default host and
// initiates the connection if available.
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
// first using the default host, then falling back to allowed hosts if needed.
func StartProxy(name string, p config.Proxy, cfg *config.Config) error {
	// Extract backend from default host
	hostName := p.Default
	if hostName == "" {
		return fmt.Errorf("no default host set for proxy '%s'", name)
	}

	hostCfg, ok := cfg.Hosts[hostName]
	if !ok {
		return fmt.Errorf("host '%s' not found in config", hostName)
	}

	backendName := hostCfg.Backend
	if backendName == "" {
		backendName = "ssh_exec"
	}

	backend, err := interfaces.GetBackend(backendName)
	if err != nil {
		return err
	}

	// Build host order: Default first, then allowed without duplication
	seen := map[string]bool{}
	tryHosts := []string{}

	if p.Default != "" {
		tryHosts = append(tryHosts, p.Default)
		seen[p.Default] = true
	}
	for _, h := range p.Allowed {
		if !seen[h] {
			tryHosts = append(tryHosts, h)
			seen[h] = true
		}
	}

	var lastErr error
	for _, hostName := range tryHosts {
		log.Printf("[proxy] Trying host '%s' for proxy '%s'", hostName, name)

		tryProxy := p
		tryProxy.Default = hostName

		if err := backend.Start(name, tryProxy, cfg); err == nil {
			log.Printf("[proxy] Proxy '%s' successfully started via '%s'", name, hostName)
			return nil
		} else {
			log.Printf("[proxy] Host '%s' failed: %v", hostName, err)
			lastErr = err
		}
	}

	return fmt.Errorf("all attempts failed for proxy '%s': %v", name, lastErr)
}
