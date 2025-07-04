// Package control provides client-side communication logic to interact with the geistd daemon.
// It sends commands via the configured Unix socket and parses the response.
package control

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const defaultSocket = "/tmp/portgeist.sock"

// listProxies connects to the geistd daemon and requests the list of configured proxies.
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
