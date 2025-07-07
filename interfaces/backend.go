// Package interfaces defines extensible interfaces for proxy backend implementations.
// Each backend (e.g., SSH, VPN, WireGuard) must implement ProxyBackend.
package interfaces

import (
	"fmt"

	"github.com/mfulz/portgeist/internal/config"
)

// ProxyBackend defines the interface for launching, stopping,
// and querying the state of a proxy tunnel.
type ProxyBackend interface {
	// Start attempts to launch a dynamic tunnel for the given proxy using
	// the configuration context provided.
	Start(name string, proxy config.Proxy, cfg *config.Config) error

	// Stop attempts to cleanly shut down the proxy identified by its name.
	Stop(name string) error

	// Status returns the runtime status of a proxy by name.
	// It returns the OS process ID (if running) and a boolean indicating whether
	// the proxy is actively tracked and presumed alive.
	//
	// If the proxy is not running or unknown to the backend, PID will be 0 and running=false.
	Status(name string) (pid int, running bool)

	// Configure allows setting backend-specific configuration parameters.
	// This is called before any Start/Stop/Status calls.
	Configure(name string, config map[string]any) error
}

// RunningInstance represents an actively running proxy instance.
// It must be stoppable via a Stop() call.
type RunningInstance interface {
	Stop()
}

// InstanceReportingBackend is an optional extension to ProxyBackend.
// It allows the backend to expose a live RunningInstance for later control.
type InstanceReportingBackend interface {
	ProxyBackend
	GetInstance(name string) RunningInstance
}

// ExitAwareBackend is an optional extension to ProxyBackend.
// It allows the backend to notify a callback when a process exits.
type ExitAwareBackend interface {
	ProxyBackend
	SetExitHandler(func(name string))
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
