// Package control provides authenticated client access to the running geistd daemon.
// It handles the communication over Unix or TCP sockets and offers high-level
// functions to manage and inspect configured proxy definitions.
package control

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"

	controlcli "github.com/mfulz/portgeist/internal/controlcli"
	"github.com/mfulz/portgeist/protocol"
)

func SendStructuredRequest(req protocol.Request) (*protocol.Response, error) {
	conf, err := controlcli.LoadClientConfig()
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	if conf.TCP != "" {
		conn, err = net.Dial("tcp", conf.TCP)
	} else {
		conn, err = net.Dial("unix", conf.Socket)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if req.Auth == nil {
		req.Auth = &protocol.Auth{
			User:  conf.User,
			Token: conf.Token,
		}
	}

	err = protocol.WriteRequest(conn, &req)
	if err != nil {
		return nil, err
	}

	return protocol.ReadResponse(conn)
}

// ProxyStatus represents the runtime status of a proxy as reported by the daemon.
type ProxyStatus struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Running bool   `json:"running"`
	PID     int    `json:"pid"`
}

// ProxyInfo represents the full configuration and runtime state of a proxy.
type ProxyInfo struct {
	Name         string   `json:"name"`
	Port         int      `json:"port"`
	Backend      string   `json:"backend"`
	Default      string   `json:"default"`
	Allowed      []string `json:"allowed"`
	Autostart    bool     `json:"autostart"`
	Running      bool     `json:"running"`
	PID          int      `json:"pid"`
	AllowedUsers []string `json:"allowed_users"`
}

// SendCommandWithAuth sends an authenticated command to the geistd daemon
// and returns the raw byte response. It automatically handles authentication
// and connection setup via Unix or TCP socket depending on the client config.
func SendCommandWithAuth(command string) ([]byte, error) {
	conf, err := controlcli.LoadClientConfig()
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	if conf.TCP != "" {
		conn, err = net.Dial("tcp", conf.TCP)
	} else {
		conn, err = net.Dial("unix", conf.Socket)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	fmt.Fprintf(conn, "auth:%s:%s\n", conf.User, conf.Token)
	fmt.Fprintf(conn, "%s\n", command)

	raw, err := io.ReadAll(conn)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(string(raw), "error:") {
		return nil, fmt.Errorf(strings.TrimSpace(string(raw)))
	}

	return raw, nil
}

// ListProxiesWithAuth returns all known proxy names as reported by the daemon.
// Requires valid authentication and access to the configured control socket.
func ListProxiesWithAuth() ([]string, error) {
	raw, err := SendCommandWithAuth("proxy list")
	if err != nil {
		return nil, err
	}
	var proxies []string
	if err := json.Unmarshal(raw, &proxies); err != nil {
		return nil, fmt.Errorf("invalid response format: %w", err)
	}
	return proxies, nil
}

// GetProxyStatusWithAuth queries the daemon for the runtime state (PID, running, backend)
// of a specific proxy identified by name. Returns an error if access is denied or the proxy
// is unknown.
func GetProxyStatusWithAuth(name string) (*ProxyStatus, error) {
	raw, err := SendCommandWithAuth(fmt.Sprintf("proxy status %s", name))
	if err != nil {
		return nil, err
	}
	var status ProxyStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &status, nil
}

// GetProxyInfoWithAuth retrieves the full proxy configuration and its current
// runtime state from the daemon. This includes port, backend, allowed hosts/users,
// autostart flag, and process information.
func GetProxyInfoWithAuth(name string) (*ProxyInfo, error) {
	raw, err := SendCommandWithAuth(fmt.Sprintf("proxy info %s", name))
	if err != nil {
		return nil, err
	}
	var info ProxyInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &info, nil
}
