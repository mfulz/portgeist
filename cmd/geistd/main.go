// Command geistd is the main entry point for the Portgeist daemon.
// It loads configuration, initializes the control interface (unix/tcp),
// and starts all proxies marked for autostart in the config file.
// On termination signals, it gracefully shuts down all running proxies.
package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mfulz/portgeist/dispatch"
	_ "github.com/mfulz/portgeist/internal/backend"
	"github.com/mfulz/portgeist/internal/control"
	"github.com/mfulz/portgeist/protocol"

	"github.com/mfulz/portgeist/interfaces"
	"github.com/mfulz/portgeist/internal/config"
	"github.com/mfulz/portgeist/internal/proxy"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[geistd] Failed to load config: %v", err)
	}

	log.Println("[geistd] Configuration loaded successfully")
	// Global dispatcher registry
	dispatcher := dispatch.New()

	dispatcher.Register(protocol.CmdProxyStart, func(req *protocol.Request) *protocol.Response {
		var payload protocol.StartRequest
		data, _ := json.Marshal(req.Data)
		if err := json.Unmarshal(data, &payload); err != nil {
			return &protocol.Response{Status: "error", Error: "invalid start payload"}
		}

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := "unauthenticated"
		if req.Auth != nil {
			user = req.Auth.User
		}
		if !control.IsControlAllowed(proxyCfg, user, !cfg.Control.Auth.Enabled) {
			return &protocol.Response{Status: "error", Error: "access denied"}
		}

		if err := proxy.StartProxy(payload.Name, proxyCfg, cfg); err != nil {
			return &protocol.Response{Status: "error", Error: err.Error()}
		}
		return &protocol.Response{Status: "ok"}
	})

	dispatcher.Register(protocol.CmdProxyStop, func(req *protocol.Request) *protocol.Response {
		var payload protocol.StopRequest
		data, _ := json.Marshal(req.Data)
		if err := json.Unmarshal(data, &payload); err != nil {
			return &protocol.Response{Status: "error", Error: "invalid stop payload"}
		}

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := "unauthenticated"
		if req.Auth != nil {
			user = req.Auth.User
		}
		if !control.IsControlAllowed(proxyCfg, user, !cfg.Control.Auth.Enabled) {
			return &protocol.Response{Status: "error", Error: "access denied"}
		}

		if err := proxy.StopProxy(payload.Name, proxyCfg, cfg); err != nil {
			return &protocol.Response{Status: "error", Error: err.Error()}
		}

		return &protocol.Response{Status: "ok"}
	})

	dispatcher.Register(protocol.CmdProxyStatus, func(req *protocol.Request) *protocol.Response {
		var payload protocol.StatusRequest
		data, _ := json.Marshal(req.Data)
		if err := json.Unmarshal(data, &payload); err != nil {
			return &protocol.Response{Status: "error", Error: "invalid status payload"}
		}

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := "unauthenticated"
		if req.Auth != nil {
			user = req.Auth.User
		}
		if !control.IsControlAllowed(proxyCfg, user, !cfg.Control.Auth.Enabled) {
			return &protocol.Response{Status: "error", Error: "access denied"}
		}

		status, err := proxy.GetProxyStatus(payload.Name, proxyCfg, cfg)
		if err != nil {
			return &protocol.Response{Status: "error", Error: err.Error()}
		}

		return &protocol.Response{Status: "ok", Data: status}
	})

	dispatcher.Register(protocol.CmdProxyList, func(req *protocol.Request) *protocol.Response {
		user := "unauthenticated"
		if req.Auth != nil {
			user = req.Auth.User
		}

		var result []string
		for name, proxyCfg := range cfg.Proxies.Proxies {
			if control.IsControlAllowed(proxyCfg, user, !cfg.Control.Auth.Enabled) {
				result = append(result, name)
			}
		}

		return &protocol.Response{
			Status: "ok",
			Data:   result,
		}
	})

	dispatcher.Register(protocol.CmdProxyInfo, func(req *protocol.Request) *protocol.Response {
		var payload protocol.InfoRequest
		data, _ := json.Marshal(req.Data)
		if err := json.Unmarshal(data, &payload); err != nil {
			return &protocol.Response{Status: "error", Error: "invalid info payload"}
		}

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := "unauthenticated"
		if req.Auth != nil {
			user = req.Auth.User
		}
		if !control.IsControlAllowed(proxyCfg, user, !cfg.Control.Auth.Enabled) {
			return &protocol.Response{Status: "error", Error: "access denied"}
		}

		hostCfg, ok := cfg.Hosts[proxyCfg.Default]
		if !ok {
			return &protocol.Response{Status: "error", Error: "host not found"}
		}
		backend := hostCfg.Backend
		if backend == "" {
			backend = "ssh_exec"
		}

		be, err := interfaces.GetBackend(backend)
		if err != nil {
			return &protocol.Response{Status: "error", Error: err.Error()}
		}
		pid, running := be.Status(payload.Name)

		resp := &protocol.InfoResponse{
			Name:    payload.Name,
			Backend: backend,
			Host:    hostCfg.Address,
			Port:    hostCfg.Port,
			Login:   hostCfg.Login,
			Running: running,
			PID:     pid,
			Allowed: proxyCfg.AllowedControls,
		}
		return &protocol.Response{Status: "ok", Data: resp}
	})

	// Übergib Dispatcher an den Server
	control.SetDispatcher(dispatcher)

	if err := control.StartServer(cfg); err != nil {
		log.Fatalf("[geistd] Control interface failed: %v", err)
	}

	err = proxy.StartAutostartProxies(cfg)
	if err != nil {
		log.Printf("[geistd] Error starting proxies: %v", err)
	}

	log.Println("[geistd] Daemon is running. Waiting for control events...")

	// Handle termination signals to shut down proxies cleanly
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan // wait for signal

	log.Println("[geistd] Termination signal received. Stopping proxies...")

	for name, p := range cfg.Proxies.Proxies {
		hostCfg, ok := cfg.Hosts[p.Default]
		if !ok {
			log.Printf("[geistd] Host '%s' not found for proxy '%s'", p.Default, name)
			continue
		}
		backendName := hostCfg.Backend
		if backendName == "" {
			backendName = "ssh_exec"
		}

		backend, err := interfaces.GetBackend(backendName)
		if err != nil {
			log.Printf("[geistd] Unknown backend for proxy '%s': %v", name, err)
			continue
		}

		if err := backend.Stop(name); err != nil {
			log.Printf("[geistd] Failed to stop proxy '%s': %v", name, err)
		}
	}

	log.Println("[geistd] Shutdown complete. Exiting.")
}
