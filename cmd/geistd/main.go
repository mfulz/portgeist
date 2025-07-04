// Command geistd is the main entry point for the Portgeist daemon.
// It loads configuration, initializes the control interface (unix/tcp),
// and starts all proxies marked for autostart in the config file.
// On termination signals, it gracefully shuts down all running proxies.
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mfulz/portgeist/internal/backend"

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
		backendName := p.Backend
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
