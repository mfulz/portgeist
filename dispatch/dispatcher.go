// Package dispatch provides a central registry and dispatcher for protocol-based
// commands handled by the portgeist daemon.
package dispatch

import (
	"sync"

	"github.com/mfulz/portgeist/protocol"
)

// HandlerFunc defines the signature of a command handler.
type HandlerFunc func(req *protocol.Request) *protocol.Response

// Dispatcher maps command strings to their handlers.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]HandlerFunc
}

// New creates a new Dispatcher.
func New() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]HandlerFunc),
	}
}

// Register binds a command string to a handler.
func (d *Dispatcher) Register(command string, handler HandlerFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[command] = handler
}

// Dispatch executes the handler for a given request.
func (d *Dispatcher) Dispatch(req *protocol.Request) *protocol.Response {
	d.mu.RLock()
	handler, ok := d.handlers[req.Type]
	d.mu.RUnlock()

	if !ok {
		return &protocol.Response{
			Status: "error",
			Error:  "unknown command",
		}
	}

	return handler(req)
}
