// Package protocol defines the message structures and types used for communication
// between geistctl and the geistd daemon. It can be used externally to build
// additional tooling or integrations.
package protocol

// Command types for Request.Type
const (
	CmdProxyStart     = "proxy.start"
	CmdProxyStop      = "proxy.stop"
	CmdProxyStatus    = "proxy.status"
	CmdProxyList      = "proxy.list"
	CmdProxyInfo      = "proxy.info"
	CmdProxySetActive = "proxy.setactive"
	CmdPing           = "system.ping"
	CmdProxyResolv    = "proxy.resolve"
)

// Request represents a message sent from a client to the daemon.
type Request struct {
	Type string      `json:"type"`           // e.g. "proxy.start", "proxy.status"
	Auth *Auth       `json:"auth,omitempty"` // Optional auth block
	Data interface{} `json:"data,omitempty"` // Optional payload
}

// Response represents a message sent from the daemon to a client.
type Response struct {
	Status string      `json:"status"`          // "ok" or "error"
	Data   interface{} `json:"data,omitempty"`  // Optional result
	Error  string      `json:"error,omitempty"` // Optional error message
}

// Auth holds authentication information for a client.
type Auth struct {
	User  string `json:"user"`
	Token string `json:"token"`
}

// --- Payload Types ---

type StartRequest struct {
	Name string `json:"name"`
}

type StopRequest struct {
	Name string `json:"name"`
}

type StatusRequest struct {
	Name string `json:"name"`
}

type InfoRequest struct {
	Name string `json:"name"`
}

type StatusResponse struct {
	Name       string `json:"name"`
	Backend    string `json:"backend"`
	Running    bool   `json:"running"`
	PID        int    `json:"pid"`
	ActiveHost string `json:"active_host"`
}

// InfoResponse combines proxy config and runtime status.
type InfoResponse struct {
	Name       string `json:"name"`
	Backend    string `json:"backend"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Login      string `json:"login"`
	Running    bool   `json:"running"`
	PID        int    `json:"pid"`
	ActiveHost string `json:"active_host"`
}

// SetActiveRequest sets the active host for a proxy.
type SetActiveRequest struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

// ListResponse wraps a list of available proxy names for structured parsing.
type ListResponse struct {
	Proxies []string `json:"proxies"`
}

type ResolvRequest struct {
	Alias string `json:"alias"`
}

type ResolvResponse struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}
