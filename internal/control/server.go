// Package control provides the server-side daemon logic to accept CLI commands
// via a Unix socket and respond to client requests like 'proxy list'.
package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/mfulz/portgeist/dispatch"
	"github.com/mfulz/portgeist/internal/acl"
	"github.com/mfulz/portgeist/internal/configd"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/mfulz/portgeist/protocol"
)

var dispatcher *dispatch.Dispatcher

func SetDispatcher(d *dispatch.Dispatcher) {
	dispatcher = d
}

// StartServerInstance starts a control listener based on the given configuration.
// Supports "unix" and "tcp" control modes.
func StartServerInstance(inst configd.ControlInstance, cfg *configd.Config) error {
	var ln net.Listener
	var err error

	switch inst.Mode {
	case "unix":
		_ = os.Remove(inst.Listen) // Remove stale socket
		ln, err = net.Listen("unix", inst.Listen)
	case "tcp":
		ln, err = net.Listen("tcp", inst.Listen)
	default:
		return fmt.Errorf("unsupported control mode: %s", inst.Mode)
	}

	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				logging.Log.Infof("[control:%s] Accept error: %v", inst.Name, err)
				continue
			}
			go handleConn(conn, inst, cfg)
		}
	}()

	return nil
}

// handleConn handles an individual control connection.
// It reads a JSON-encoded protocol.Request from the connection,
// dispatches it via the global dispatcher, and writes the JSON response.
func handleConn(conn net.Conn, inst configd.ControlInstance, cfg *configd.Config) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var req protocol.Request
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				logging.Log.Infof("[control:%s] Client closed connection early", inst.Name)
				return
			}
			logging.Log.Infof("[control:%s] Failed to decode request: %v", inst.Name, err)
			return
		}

		// user, ok := Authenticate(&req, cfg, inst.Auth.Enabled)
		if !acl.Authenticate(req.Auth) {
			// if !ok {
			logging.Log.Infof("[control:%s] Invalid credentials for user: %s", inst.Name, req.Auth.User)
			_ = encoder.Encode(&protocol.Response{
				Status: "error",
				Error:  "invalid credentials",
			})
			continue
		}

		if req.Auth == nil {
			req.Auth = &protocol.Auth{}
			req.Auth.User = "anon"
		}
		resp := dispatcher.Dispatch(&req)
		if err := encoder.Encode(resp); err != nil {
			logging.Log.Infof("[control:%s] Failed to send response: %v", inst.Name, err)
			return
		}
	}
}
