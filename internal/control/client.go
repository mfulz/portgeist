// Package control provides client-side communication logic to interact with the geistd daemon.
// It sends commands via the configured Unix socket and parses the response.
package control

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mfulz/portgeist/internal/controlcli"
)

const defaultSocket = "/tmp/portgeist.sock"

type ProxyStatus struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Running bool   `json:"running"`
	PID     int    `json:"pid"`
}

func GetProxyStatusWithAuth(name string) (*ProxyStatus, error) {
	conf, err := controlcli.LoadClientConfig()
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial("unix", conf.Socket)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Auth & Command
	fmt.Fprintf(conn, "auth:%s:%s\n", conf.User, conf.Token)
	fmt.Fprintf(conn, "proxy status %s\n", name)

	reader := bufio.NewReader(conn)
	raw, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(string(raw), "error:") {
		return nil, fmt.Errorf(strings.TrimSpace(string(raw)))
	}

	var status ProxyStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &status, nil
}

func GetProxyStatus(name string) (*ProxyStatus, error) {
	conn, err := net.Dial("unix", defaultSocket)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "proxy status %s\n", name)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	raw, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var status ProxyStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &status, nil
}

// ListProxies connects to the geistd daemon and requests the list of configured proxies.
func ListProxies() ([]string, error) {
	conn, err := net.DialTimeout("unix", defaultSocket, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not connect to daemon socket: %w", err)
	}
	defer conn.Close()

	// Send command
	_, err = fmt.Fprintln(conn, "proxy list")
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	raw, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Expecting JSON array of strings
	var proxies []string
	if err := json.Unmarshal(raw, &proxies); err != nil {
		return nil, fmt.Errorf("invalid response format: %w", err)
	}

	return proxies, nil
}

// SendCommand sends a raw command over the control socket and returns any error.
func SendCommand(cmd string) error {
	conn, err := net.Dial("unix", defaultSocket)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = fmt.Fprintln(conn, cmd)
	return err
}

func SendCommandWithAuth(cmd string) error {
	conf, err := controlcli.LoadClientConfig()
	if err != nil {
		return err
	}

	conn, err := net.Dial("unix", conf.Socket)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Auth line
	fmt.Fprintf(conn, "auth:%s:%s\n", conf.User, conf.Token)
	// Command
	fmt.Fprintf(conn, "%s\n", cmd)

	return nil
}
