// Package controlcli handles daemon communication and request encoding from geistctl.
package controlcli

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/mfulz/portgeist/protocol"
)

// SendCommandWithAuth connects to a selected daemon and sends a request with authentication.
func SendCommandWithAuth(cfg *CTLConfig, daemonName, userName, command string, data interface{}) (*protocol.Response, error) {
	daemon, ok := cfg.Daemons[daemonName]
	if !ok {
		return nil, fmt.Errorf("daemon '%s' not found", daemonName)
	}

	user, ok := cfg.Users[userName]
	if !ok {
		return nil, fmt.Errorf("user '%s' not found", userName)
	}

	req := protocol.Request{
		Type: command,
		Data: data,
		Auth: &protocol.Auth{
			User:  userName,
			Token: user.Token,
		},
	}

	var conn net.Conn
	var err error

	if daemon.Socket != "" {
		conn, err = net.DialTimeout("unix", daemon.Socket, 2*time.Second)
	} else if daemon.TCP != "" {
		conn, err = net.DialTimeout("tcp", daemon.TCP, 2*time.Second)
	} else {
		return nil, fmt.Errorf("invalid daemon config: no socket or tcp defined")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon '%s': %w", daemonName, err)
	}

	// Achtung: Close erst NACH erfolgreichem Connect & Encode setzen
	// defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	if err := enc.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer conn.Close()

	var resp protocol.Response
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &resp, nil
}

// connectToDaemon establishes a connection to the Portgeist daemon using either
// a Unix socket or a TCP address, depending on the mode specified.
// It returns a net.Conn ready for protocol exchange.
//
// Valid modes:
// - "unix": uses a local Unix domain socket (addr = path to socket file)
// - "tcp": connects to a remote daemon over TCP (addr = host:port)
func connectToDaemon(mode, addr string) (net.Conn, error) {
	switch mode {
	case "unix":
		return net.Dial("unix", addr)
	case "tcp":
		return net.Dial("tcp", addr)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", mode)
	}
}

// SendDirectCommand sends a request directly to a daemon using a raw address
// (either UNIX socket path or TCP host:port), bypassing any configured client mappings.
// This is used for ad-hoc communication with daemons not listed in the ctl_config.
func SendDirectCommand(addr, token, user, command string, payload interface{}) (*protocol.Response, error) {
	var mode string
	if addr == "" {
		return nil, fmt.Errorf("address required")
	}

	if len(addr) > 0 && addr[0] == '/' {
		mode = "unix"
	} else {
		mode = "tcp"
	}

	conn, err := connectToDaemon(mode, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	req := &protocol.Request{
		Type: command,
		Auth: &protocol.Auth{
			User:  user,
			Token: token,
		},
		Data: payload,
	}

	if err := protocol.Encode(conn, req); err != nil {
		return nil, err
	}

	resp, err := protocol.Decode(conn)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ListAvailableDaemons returns a list of configured daemon names.
func ListAvailableDaemons(cfg *CTLConfig) []string {
	var list []string
	for name := range cfg.Daemons {
		list = append(list, name)
	}
	return list
}

// GuessDefaultDaemon returns the first daemon name or empty string if none configured.
func GuessDefaultDaemon(cfg *CTLConfig) string {
	for name := range cfg.Daemons {
		return name
	}
	return ""
}
