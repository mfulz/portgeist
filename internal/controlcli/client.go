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
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	if err := enc.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var resp protocol.Response
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &resp, nil
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
