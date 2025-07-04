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

func StartServer(cfg *config.Config) error {
	if cfg.Control.Mode != "unix" {
		return fmt.Errorf("only unix control mode supported for now")
	}

	// Clean up existing socket
	if _, err := os.Stat(cfg.Control.Socket); err == nil {
		_ = os.Remove(cfg.Control.Socket)
	}

	listener, err := net.Listen("unix", cfg.Control.Socket)
	if err != nil {
		return fmt.Errorf("failed to bind unix socket: %w", err)
	}

	log.Printf("[control] Listening on Unix socket: %s", cfg.Control.Socket)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("[control] Accept error: %v", err)
				continue
			}
			go handleConn(conn, cfg)
		}
	}()

	// Optional TCP listen
	if cfg.Control.Listen != "" {
		go func() {
			tcpLn, err := net.Listen("tcp", cfg.Control.Listen)
			if err != nil {
				log.Fatalf("[control] TCP listen failed: %v", err)
			}
			log.Printf("[control] Listening on TCP: %s", cfg.Control.Listen)
			for {
				conn, err := tcpLn.Accept()
				if err != nil {
					log.Printf("[control] TCP accept error: %v", err)
					continue
				}
				go handleConn(conn, cfg)
			}
		}()
	}

	return nil
}

func handleConn(conn net.Conn, cfg *config.Config) {
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
