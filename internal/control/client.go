// Package control provides authenticated client access to the running geistd daemon.
// It handles the communication over Unix or TCP sockets and offers high-level
// functions to manage and inspect configured proxy definitions.
package control

import (
	"net"

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
