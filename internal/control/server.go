// Package control provides the server-side daemon logic to accept CLI commands
// via a Unix socket and respond to client requests like 'proxy list'.
package control

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/mfulz/portgeist/dispatch"
	"github.com/mfulz/portgeist/internal/config"
	"github.com/mfulz/portgeist/protocol"
)

var dispatcher *dispatch.Dispatcher

func SetDispatcher(d *dispatch.Dispatcher) {
	dispatcher = d
}

// StartServerInstance starts a control listener based on the given configuration.
// Supports "unix" and "tcp" control modes.
func StartServerInstance(inst config.ControlInstance, cfg *config.Config) error {
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
				log.Printf("[control:%s] Accept error: %v", inst.Name, err)
				continue
			}
			go handleConn(conn, inst, cfg)
		}
	}()

	return nil
}

func handleConn(conn net.Conn, inst config.ControlInstance, cfg *config.Config) {
	defer conn.Close()

	req, err := protocol.ReadRequest(conn)
	if err != nil {
		protocol.WriteResponse(conn, &protocol.Response{
			Status: "error",
			Error:  fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Dispatch über zentrale Registry
	resp := dispatcher.Dispatch(req)
	_ = protocol.WriteResponse(conn, resp)
}

func handleProxyCmd(conn net.Conn, name, user string, cfg *config.Config, skip bool, fn func(string, config.Proxy, *config.Config) error) {
	proxyCfg, ok := cfg.Proxies.Proxies[name]
	if !ok {
		conn.Write([]byte("error: unknown proxy\n"))
		return
	}
	if !IsControlAllowed(proxyCfg, user, skip) {
		conn.Write([]byte("error: access denied\n"))
		return
	}
	if err := fn(name, proxyCfg, cfg); err != nil {
		conn.Write([]byte(fmt.Sprintf("error: %v\n", err)))
	} else {
		conn.Write([]byte("ok\n"))
	}
}
