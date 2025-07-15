// Package main is the entry point for the geistd daemon.
// It initializes proxies and control interfaces based on the configuration.
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mfulz/portgeist/dispatch"
	"github.com/mfulz/portgeist/internal/acl"
	_ "github.com/mfulz/portgeist/internal/backend"
	"github.com/mfulz/portgeist/internal/configd"
	"github.com/mfulz/portgeist/internal/configloader"
	"github.com/mfulz/portgeist/internal/control"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/mfulz/portgeist/internal/proxy"
	"github.com/mfulz/portgeist/protocol"
)

func main() {
	err := configd.LoadConfig()
	if err != nil {
		logging.Log.Fatalf("[geistd] Failed to load config: %v", err)
	}
	cfg := configloader.MustGetConfig[*configd.Config]()
	logging.Log.Debugln("[geistd] Configuration loaded successfully:\n%v", cfg)

	if err := acl.Init(cfg.ACL, []acl.Permission{
		"proxy_start",
		"proxy_stop",
		"proxy_status",
		"proxy_list",
		"proxy_info",
		"proxy_setactive",
		"proxy_resolve",
	}); err != nil {
		logging.Log.Fatalf("[geistd] Failed to init acls: %v", err)
	}

	// Start autostart proxies
	for name, p := range cfg.Proxies.Proxies {
		if p.Autostart {
			logging.Log.Infof("[proxy] Autostart enabled for '%s'", name)
			if err := proxy.StartProxy(name, p, cfg); err != nil {
				logging.Log.Warnf("[proxy] Failed to start '%s': %v", name, err)
			} else {
				logging.Log.Infof("[proxy] Proxy '%s' started", name)
			}
		}
	}

	// Start all enabled control instances
	for _, inst := range cfg.Control.Instances {
		if !inst.Enabled {
			continue
		}

		go func(inst configd.ControlInstance) {
			logging.Log.Infof("[control:%s] Starting (%s): %s", inst.Name, inst.Mode, inst.Listen)

			dispatcher := dispatch.New()
			dispatcher.Register(protocol.CmdProxyStart, control.StartProxyHandler(cfg, inst))
			dispatcher.Register(protocol.CmdProxyStop, control.StopProxyHandler(cfg, inst))
			dispatcher.Register(protocol.CmdProxyStatus, control.ProxyStatusHandler(cfg, inst))
			dispatcher.Register(protocol.CmdProxyList, control.ProxyListHandler(cfg, inst))
			dispatcher.Register(protocol.CmdProxyInfo, control.ProxyInfoHandler(cfg, inst))
			dispatcher.Register(protocol.CmdProxySetActive, control.ProxySetActiveHandler(cfg, inst))
			dispatcher.Register(protocol.CmdProxyResolv, control.ResolveProxyHandler(cfg, inst))
			control.SetDispatcher(dispatcher)

			if err := control.StartServerInstance(inst, cfg); err != nil {
				logging.Log.Errorf("[control:%s] Error: %v", inst.Name, err)
			}
		}(inst)
	}

	logging.Log.Infoln("[geistd] Daemon is running. Waiting for control events...")
	waitForShutdown()
	// select {}
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logging.Log.Infof("[geistd] Caught signal: %s. Shutting down...", sig)

	proxy.StopAll()

	os.Exit(0)
}
