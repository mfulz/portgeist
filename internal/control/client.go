// Package control provides authenticated client access to the running geistd daemon.
// It handles the communication over Unix or TCP sockets and offers high-level
// functions to manage and inspect configured proxy definitions.
package control

// ProxyStatus represents the runtime status of a proxy as reported by the daemon.
type ProxyStatus struct {
	Name       string `json:"name"`
	Backend    string `json:"backend"`
	Running    bool   `json:"running"`
	PID        int    `json:"pid"`
	ActiveHost string `json:"active_host`
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
	ActiveHost   string   `json:"active_host`
}
