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

	"github.com/mfulz/portgeist/interfaces"
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
	reader := bufio.NewReader(conn)

	authLine, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("[control] Failed to read auth line: %v", err)
		return
	}
	authLine = strings.TrimSpace(authLine)

	var authedUser string
	skipAuthChecks := false

	if cfg.Control.Auth.Enabled {
		if !strings.HasPrefix(authLine, "auth:") {
			conn.Write([]byte("error: authentication required\n"))
			return
		}

		parts := strings.SplitN(strings.TrimPrefix(authLine, "auth:"), ":", 2)
		if len(parts) != 2 {
			conn.Write([]byte("error: malformed auth\n"))
			return
		}

		user, token := parts[0], parts[1]
		entry, ok := cfg.Control.Logins[user]
		if !ok || entry.Token != token {
			conn.Write([]byte("error: invalid credentials\n"))
			return
		}

		log.Printf("[control] Auth success for user '%s'", user)
		authedUser = user

	} else {
		// Auth disabled → accept any auth line
		authedUser = "unauthenticated"
		skipAuthChecks = true
	}

	// Read actual command
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("[control] Failed to read command: %v", err)
		return
	}

	cmd := strings.TrimSpace(line)

	// Command handlers
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
		handleProxyCmd(conn, name, authedUser, cfg, skipAuthChecks, proxy.StartProxy)

	} else if strings.HasPrefix(cmd, "proxy stop ") {
		name := strings.TrimPrefix(cmd, "proxy stop ")
		handleProxyCmd(conn, name, authedUser, cfg, skipAuthChecks, func(n string, p config.Proxy, c *config.Config) error {
			bname := p.Backend
			if bname == "" {
				bname = "ssh_exec"
			}
			backend, err := interfaces.GetBackend(bname)
			if err != nil {
				return err
			}
			return backend.Stop(n)
		})
	} else if strings.HasPrefix(cmd, "proxy status ") {
		name := strings.TrimPrefix(cmd, "proxy status ")
		if proxyCfg, ok := cfg.Proxies.Proxies[name]; ok {
			if !isControlAllowed(proxyCfg, authedUser, skipAuthChecks) {
				conn.Write([]byte("error: access denied\n"))
				return
			}
			backendName := proxyCfg.Backend
			if backendName == "" {
				backendName = "ssh_exec"
			}
			backend, err := interfaces.GetBackend(backendName)
			if err != nil {
				conn.Write([]byte("null\n"))
				return
			}
			if statusable, ok := backend.(interface {
				Status(name string) (int, bool)
			}); ok {
				pid, running := statusable.Status(name)
				resp := map[string]interface{}{
					"name":    name,
					"backend": backendName,
					"running": running,
					"pid":     pid,
				}
				out, _ := json.Marshal(resp)
				conn.Write(out)
				conn.Write([]byte("\n"))
				return
			}
		}
		conn.Write([]byte("null\n"))
	} else if strings.HasPrefix(cmd, "proxy info ") {
		name := strings.TrimPrefix(cmd, "proxy info ")
		if proxyCfg, ok := cfg.Proxies.Proxies[name]; ok {
			if !isControlAllowed(proxyCfg, authedUser, skipAuthChecks) {
				conn.Write([]byte("error: access denied\n"))
				return
			}
			backendName := proxyCfg.Backend
			if backendName == "" {
				backendName = "ssh_exec"
			}
			backend, err := interfaces.GetBackend(backendName)
			if err != nil {
				conn.Write([]byte("error: backend error\n"))
				return
			}
			pid := 0
			running := false
			if statusable, ok := backend.(interface {
				Status(name string) (int, bool)
			}); ok {
				pid, running = statusable.Status(name)
			}

			out := map[string]interface{}{
				"name":          name,
				"port":          proxyCfg.Port,
				"backend":       backendName,
				"default":       proxyCfg.Default,
				"allowed":       proxyCfg.Allowed,
				"autostart":     proxyCfg.Autostart,
				"running":       running,
				"pid":           pid,
				"allowed_users": proxyCfg.AllowedControls,
			}
			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				conn.Write([]byte("error: json marshal failed\n"))
				return
			}
			conn.Write(data)
			conn.Write([]byte("\n"))
			return
		}
		conn.Write([]byte("error: unknown proxy\n"))
	} else if strings.HasPrefix(cmd, "proxy setactive ") {
		parts := strings.Split(cmd, " ")
		if len(parts) != 4 {
			conn.Write([]byte("error: invalid setactive syntax\n"))
			return
		}
		name, target := parts[2], parts[3]
		proxyCfg, ok := cfg.Proxies.Proxies[name]
		if !ok {
			conn.Write([]byte("error: proxy not found\n"))
			return
		}
		if !isControlAllowed(proxyCfg, authedUser, skipAuthChecks) {
			conn.Write([]byte("error: access denied\n"))
			return
		}
		found := false
		for _, h := range proxyCfg.Allowed {
			if h == target {
				found = true
				break
			}
		}
		if !found {
			conn.Write([]byte("error: host not allowed\n"))
			return
		}
		// Stop and restart with new default
		backendName := proxyCfg.Backend
		if backendName == "" {
			backendName = "ssh_exec"
		}
		backend, err := interfaces.GetBackend(backendName)
		if err != nil {
			conn.Write([]byte("error: backend error\n"))
			return
		}
		err = backend.Stop(name)
		if err != nil {
			conn.Write([]byte("error: stop failed\n"))
			return
		}

		proxyCfg.Default = target
		cfg.Proxies.Proxies[name] = proxyCfg
		err = proxy.StartProxy(name, proxyCfg, cfg)
		if err != nil {
			conn.Write([]byte(fmt.Sprintf("error: start failed: %v\n", err)))
			return
		}
		conn.Write([]byte("ok\n"))
	} else {
		conn.Write([]byte("unknown command\n"))
	}
}

func isControlAllowed(proxyCfg config.Proxy, user string, skip bool) bool {
	if skip {
		return true
	}
	for _, u := range proxyCfg.AllowedControls {
		if u == user {
			return true
		}
	}
	return false
}

func handleProxyCmd(conn net.Conn, name, user string, cfg *config.Config, skip bool, fn func(string, config.Proxy, *config.Config) error) {
	proxyCfg, ok := cfg.Proxies.Proxies[name]
	if !ok {
		conn.Write([]byte("error: unknown proxy\n"))
		return
	}
	if !isControlAllowed(proxyCfg, user, skip) {
		conn.Write([]byte("error: access denied\n"))
		return
	}
	if err := fn(name, proxyCfg, cfg); err != nil {
		conn.Write([]byte(fmt.Sprintf("error: %v\n", err)))
	} else {
		conn.Write([]byte("ok\n"))
	}
}
