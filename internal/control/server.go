// Package control provides the server-side daemon logic to accept CLI commands
// via a Unix socket and respond to client requests like 'proxy list'.
package control

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/mfulz/portgeist/internal/config"
	"github.com/mfulz/portgeist/internal/proxy"
)

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

	return nil
}

func handleConn(conn net.Conn, cfg *config.Config) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("[control] Failed to read: %v", err)
		return
	}

	cmd := strings.TrimSpace(line)

	if cmd == "proxy list" {
		names := make([]string, 0, len(cfg.Proxies.Proxies))
		for name := range cfg.Proxies.Proxies {
			names = append(names, name)
		}
		resp, _ := json.Marshal(names)
		conn.Write(resp)
		conn.Write([]byte("\n"))

	} else if strings.HasPrefix(cmd, "proxy start ") {
		name := strings.TrimPrefix(cmd, "proxy start ")
		if proxyCfg, ok := cfg.Proxies.Proxies[name]; ok {
			err := proxy.StartProxy(name, proxyCfg, cfg)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("error: %v\n", err)))
			} else {
				conn.Write([]byte("ok\n"))
			}
		} else {
			conn.Write([]byte("error: unknown proxy\n"))
		}

	} else {
		conn.Write([]byte("unknown command\n"))
	}
}
