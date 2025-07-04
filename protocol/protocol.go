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
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Running bool   `json:"running"`
	PID     int    `json:"pid"`
}

// InfoResponse combines proxy config and runtime status.
type InfoResponse struct {
	Name    string   `json:"name"`
	Backend string   `json:"backend"`
	Host    string   `json:"host"`
	Port    int      `json:"port"`
	Login   string   `json:"login"`
	Running bool     `json:"running"`
	PID     int      `json:"pid"`
	Allowed []string `json:"allowed"`
}

// SetActiveRequest sets the active host for a proxy.
type SetActiveRequest struct {
	Name string `json:"name"`
	Host string `json:"host"`
}
