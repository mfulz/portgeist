// Package interfaces defines extensible interfaces for proxy backend implementations.
// Each backend (e.g., SSH, VPN, WireGuard) must implement ProxyBackend.
package interfaces

import (
	"fmt"

	"github.com/mfulz/portgeist/internal/config"
)

// ProxyBackend defines the interface for launching and stopping a proxy tunnel.
type ProxyBackend interface {
	// Start attempts to launch a dynamic tunnel for the given proxy using
	// the configuration context provided.
	Start(name string, proxy config.Proxy, cfg *config.Config) error

	// Stop attempts to cleanly shut down the proxy identified by its name.
	// May be a no-op or return an error if the backend is state-less or unsupported.
	Stop(name string) error
}

var registeredBackends = make(map[string]ProxyBackend)

// RegisterBackend adds a new backend to the global registry under a unique name.
func RegisterBackend(name string, backend ProxyBackend) {
	if _, exists := registeredBackends[name]; exists {
		panic(fmt.Sprintf("backend already registered: %s", name))
	}
	registeredBackends[name] = backend
}

// GetBackend retrieves a previously registered backend by name.
func GetBackend(name string) (ProxyBackend, error) {
	b, ok := registeredBackends[name]
	if !ok {
		return nil, fmt.Errorf("no backend registered with name: %s", name)
	}
	return b, nil
}
