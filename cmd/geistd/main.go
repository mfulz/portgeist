// Package main is the entry point for the geistd daemon.
// It initializes proxies and control interfaces based on the configuration.
package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mfulz/portgeist/dispatch"
	_ "github.com/mfulz/portgeist/internal/backend"
	"github.com/mfulz/portgeist/internal/config"
	"github.com/mfulz/portgeist/internal/configloader"
	"github.com/mfulz/portgeist/internal/control"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/mfulz/portgeist/internal/proxy"
	"github.com/mfulz/portgeist/protocol"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[geistd] Failed to load config: %v", err)
	}
	log.Println("[geistd] Configuration loaded successfully")

	cfg := configloader.MustGetConfig[*config.Config]()

	err = logging.Init()
	if err != nil {
		log.Fatalf("[geistd] Failed to load log config: %v", err)
	}
	logging.Log.Infof("[geistd] Log Config: %v", cfg.Logger)

	// Start autostart proxies
	for name, p := range cfg.Proxies.Proxies {
		if p.Autostart {
			logging.Log.Infof("[proxy] Autostart enabled for '%s'", name)
			if err := proxy.StartProxy(name, p, cfg); err != nil {
				logging.Log.Warnf("[proxy] Failed to start '%s': %v", name, err)
			} else {
				log.Printf("[proxy] Proxy '%s' started", name)
			}
		}
	}

	// Start all enabled control instances
	for _, inst := range cfg.Control.Instances {
		if !inst.Enabled {
			continue
		}

		go func(inst config.ControlInstance) {
			logging.Log.Infof("[control:%s] Starting (%s): %s", inst.Name, inst.Mode, inst.Listen)

			dispatcher := dispatch.New()

			dispatcher.Register(protocol.CmdProxyStart, func(req *protocol.Request) *protocol.Response {
				var payload protocol.StartRequest
				_ = decodePayload(req.Data, &payload)

				proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
				if !ok {
					return &protocol.Response{Status: "error", Error: "unknown proxy"}
				}
				user := extractUser(req)
				if !control.IsControlAllowed(proxyCfg, user, !inst.Auth.Enabled) {
					return &protocol.Response{Status: "error", Error: "access denied"}
				}
				if err := proxy.StartProxy(payload.Name, proxyCfg, cfg); err != nil {
					return &protocol.Response{Status: "error", Error: err.Error()}
				}
				return &protocol.Response{Status: "ok"}
			})

			dispatcher.Register(protocol.CmdProxyStop, func(req *protocol.Request) *protocol.Response {
				var payload protocol.StopRequest
				_ = decodePayload(req.Data, &payload)

				proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
				if !ok {
					return &protocol.Response{Status: "error", Error: "unknown proxy"}
				}
				user := extractUser(req)
				if !control.IsControlAllowed(proxyCfg, user, !inst.Auth.Enabled) {
					return &protocol.Response{Status: "error", Error: "access denied"}
				}
				if err := proxy.StopProxy(payload.Name, proxyCfg, cfg); err != nil {
					return &protocol.Response{Status: "error", Error: err.Error()}
				}
				return &protocol.Response{Status: "ok"}
			})

			dispatcher.Register(protocol.CmdProxyStatus, func(req *protocol.Request) *protocol.Response {
				var payload protocol.StatusRequest
				_ = decodePayload(req.Data, &payload)

				proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
				if !ok {
					return &protocol.Response{Status: "error", Error: "unknown proxy"}
				}
				user := extractUser(req)
				if !control.IsControlAllowed(proxyCfg, user, !inst.Auth.Enabled) {
					return &protocol.Response{Status: "error", Error: "access denied"}
				}
				status, err := proxy.GetProxyStatus(payload.Name, proxyCfg, cfg)
				if err != nil {
					return &protocol.Response{Status: "error", Error: err.Error()}
				}
				return &protocol.Response{Status: "ok", Data: status}
			})

			dispatcher.Register(protocol.CmdProxyList, func(req *protocol.Request) *protocol.Response {
				user := extractUser(req)
				var result []string
				for name, proxyCfg := range cfg.Proxies.Proxies {
					if control.IsControlAllowed(proxyCfg, user, !inst.Auth.Enabled) {
						result = append(result, name)
					}
				}
				return &protocol.Response{Status: "ok", Data: result}
			})

			dispatcher.Register(protocol.CmdProxyInfo, func(req *protocol.Request) *protocol.Response {
				var payload protocol.InfoRequest
				_ = decodePayload(req.Data, &payload)

				proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
				if !ok {
					return &protocol.Response{Status: "error", Error: "unknown proxy"}
				}
				user := extractUser(req)
				if !control.IsControlAllowed(proxyCfg, user, !inst.Auth.Enabled) {
					return &protocol.Response{Status: "error", Error: "access denied"}
				}
				info, err := proxy.GetProxyInfo(payload.Name, proxyCfg, cfg)
				if err != nil {
					return &protocol.Response{Status: "error", Error: err.Error()}
				}
				return &protocol.Response{Status: "ok", Data: info}
			})

			dispatcher.Register(protocol.CmdProxySetActive, func(req *protocol.Request) *protocol.Response {
				var payload protocol.SetActiveRequest
				_ = decodePayload(req.Data, &payload)

				proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
				if !ok {
					return &protocol.Response{Status: "error", Error: "unknown proxy"}
				}
				user := extractUser(req)
				if !control.IsControlAllowed(proxyCfg, user, !inst.Auth.Enabled) {
					return &protocol.Response{Status: "error", Error: "access denied"}
				}
				if _, ok := cfg.Hosts[payload.Host]; !ok {
					return &protocol.Response{Status: "error", Error: "unknown host"}
				}
				proxyCfg.Default = payload.Host
				_ = proxy.StopProxy(payload.Name, proxyCfg, cfg)
				if err := proxy.StartProxy(payload.Name, proxyCfg, cfg); err != nil {
					return &protocol.Response{Status: "error", Error: err.Error()}
				}
				return &protocol.Response{Status: "ok"}
			})

			control.SetDispatcher(dispatcher)

			if err := control.StartServerInstance(inst, cfg); err != nil {
				log.Printf("[control:%s] Error: %v", inst.Name, err)
			}
		}(inst)
	}

	log.Println("[geistd] Daemon is running. Waiting for control events...")
	waitForShutdown()
	// select {}
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("[geistd] Caught signal: %s. Shutting down...", sig)

	proxy.StopAll()

	os.Exit(0)
}

// decodePayload marshals a map into the target struct.
func decodePayload(input any, out any) error {
	data, _ := json.Marshal(input)
	return json.Unmarshal(data, out)
}

// extractUser returns the request auth user or "unauthenticated".
func extractUser(req *protocol.Request) string {
	if req.Auth != nil {
		return req.Auth.User
	}
	return "unauthenticated"
}
