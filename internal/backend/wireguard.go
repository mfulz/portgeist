// Package backend provides concrete backend implementations for proxy launching.
// This file implements the WireGuard backend using the wireguard-go netstack.
// It behaves analog zu ssh_native: das Backend baut intern einen eigenen
// WireGuard-Stack auf und stellt nur einen Dialer für den SOCKS-Server bereit.
package backend

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mfulz/portgeist/interfaces"
	"github.com/mfulz/portgeist/internal/configd"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/things-go/go-socks5"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

type TunnelResolver struct {
	socks5.NameResolver
	tnet *netstack.Net
}

func NewTunnelResolver(tnet *netstack.Net) (resolver *TunnelResolver) {
	resolver = new(TunnelResolver)
	resolver.tnet = tnet
	return resolver
}

func (r TunnelResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	addrs, err := r.tnet.LookupContextHost(ctx, name)
	if err != nil {
		return nil, nil, err
	}

	naddr := len(addrs)
	if naddr == 0 {
		return nil, nil, fmt.Errorf("no address found for: %s", name)
	}

	rand.Shuffle(naddr, func(i, j int) {
		addrs[i], addrs[j] = addrs[j], addrs[i]
	})

	var addr netip.Addr
	for _, saddr := range addrs {
		addr, err = netip.ParseAddr(saddr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, nil, err
	}

	return ctx, addr.AsSlice(), nil
}

// wireguardBackend implements the ProxyBackend and ExitAwareBackend interfaces.
type wireguardBackend struct {
	mu           sync.Mutex
	instances    map[string]*wireguardInstance
	settings     map[string]map[string]any
	stopFlags    map[string]bool
	exitCallback func(name string)
}

// wireguardInstance represents a running SOCKS listener + WireGuard device.
type wireguardInstance struct {
	listener net.Listener
	dev      *device.Device
}

// Stop terminates the SOCKS listener and shuts down the WireGuard device.
func (w *wireguardInstance) Stop() {
	if w.listener != nil {
		_ = w.listener.Close()
	}
	if w.dev != nil {
		w.dev.Close()
	}
}

// init registers the WireGuard backend.
func init() {
	interfaces.RegisterBackend("wireguard", &wireguardBackend{
		instances: make(map[string]*wireguardInstance),
		settings:  make(map[string]map[string]any),
		stopFlags: make(map[string]bool),
	})
}

// SetExitHandler registers a callback for unexpected listener exits.
func (w *wireguardBackend) SetExitHandler(cb func(name string)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.exitCallback = cb
}

// Configure stores the merged configuration for a given proxy name.
func (w *wireguardBackend) Configure(name string, cfg map[string]any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if cfg == nil {
		cfg = make(map[string]any)
	}
	w.settings[name] = cfg
	return nil
}

// Start creates a WireGuard netstack instance for this proxy and launches a
// SOCKS5 listener that routes traffic via tnet.DialContext.
//
// Konfig-Flow:
//   - hostName := proxy.Default
//   - host := cfg.Hosts[hostName]
//   - wgLogin := cfg.WGLogins[host.Login]  // wglogins-Map
//   - backend-spezifische Settings (publickey, presharedkey, allowed_ips, ...)
func (w *wireguardBackend) Start(name string, p configd.Proxy, cfg *configd.Config) error {
	// Host analog ssh_nativ: p.Default → cfg.Hosts[hostName]
	hostName := p.Default
	host, ok := cfg.Hosts[hostName]
	if !ok {
		return fmt.Errorf("default host '%s' not found for proxy '%s'", hostName, name)
	}

	// WireGuard-Login über Host.Login aus wglogins-Map
	wgLogin, ok := cfg.WGLogins[host.Login]
	if !ok {
		return fmt.Errorf("wg login '%s' not found for host '%s'", host.Login, hostName)
	}

	w.mu.Lock()
	cfgMap := w.settings[name]
	w.mu.Unlock()
	if cfgMap == nil {
		cfgMap = make(map[string]any)
	}

	publicKeyStr, err := requireString(cfgMap, "publickey")
	if err != nil {
		return err
	}
	presharedKeyStr := optionalString(cfgMap, "presharedkey")
	allowedIPsStrs, err := optionalStringSlice(cfgMap, "allowed_ips")
	if err != nil {
		return err
	}
	keepaliveSeconds, err := optionalInt(cfgMap, "persistent_keepalive", 0)
	if err != nil {
		return err
	}
	connectTimeoutSeconds, err := optionalInt(cfgMap, "connect_timeout", 5)
	if err != nil {
		return err
	}

	// Endpoint aus Host: address + port
	endpoint := host.Address
	if host.Port != 0 {
		endpoint = net.JoinHostPort(host.Address, strconv.Itoa(host.Port))
	}

	// WG-Addresses (v4/v6) aus wgLogin.Address
	wgAddrs, err := parseWGAddresses(wgLogin.Address)
	if err != nil {
		return fmt.Errorf("[wireguard] invalid wglogin address for '%s': %w", host.Login, err)
	}

	// DNS-Adressen (optional; ungültige werden geskippt)
	wgDNS := parseDNSAddresses(wgLogin.DNS)

	logging.Log.Infof(
		"[wireguard] Creating netstack tunnel for proxy='%s' host='%s' login='%s' endpoint='%s' dns='%s'",
		name, hostName, host.Login, endpoint, wgDNS,
	)

	// WireGuard-Netstack-TUN erzeugen
	tunDev, tnet, err := netstack.CreateNetTUN(
		wgAddrs,
		wgDNS,
		wgLogin.MTU,
	)
	if err != nil {
		return fmt.Errorf("[wireguard] CreateNetTUN failed for proxy '%s': %w", name, err)
	}

	// WireGuard-Device mit Default-Bind + Logger
	logger := device.NewLogger(device.LogLevelError, fmt.Sprintf("wg-%s ", name))
	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), logger)

	wgLoginPrivateKey, _ := wgKeyBase64ToHex(wgLogin.PrivateKey)
	wgPublicKey, _ := wgKeyBase64ToHex(publicKeyStr)
	wgPresharedKey, _ := wgKeyBase64ToHex(presharedKeyStr)

	// IPC-Konfig für wireguard-go zusammenbauen
	var b strings.Builder
	// PrivateKey aus wglogin
	b.WriteString("private_key=")
	b.WriteString(strings.TrimSpace(wgLoginPrivateKey))
	b.WriteString("\n")

	// Peer PublicKey aus Backend-Config
	b.WriteString("public_key=")
	b.WriteString(strings.TrimSpace(wgPublicKey))
	b.WriteString("\n")

	// Optional: PSK
	if ps := strings.TrimSpace(wgPresharedKey); ps != "" {
		b.WriteString("preshared_key=")
		b.WriteString(ps)
		b.WriteString("\n")
	}

	// AllowedIPs
	for _, cidr := range allowedIPsStrs {
		c := strings.TrimSpace(cidr)
		if c == "" {
			continue
		}
		b.WriteString("allowed_ip=")
		b.WriteString(c)
		b.WriteString("\n")
	}

	// Endpoint (AirVPN-Server, etc.)
	b.WriteString("endpoint=")
	b.WriteString(endpoint)
	b.WriteString("\n")

	// Optional Keepalive
	if keepaliveSeconds > 0 {
		b.WriteString("persistent_keepalive_interval=")
		b.WriteString(strconv.Itoa(keepaliveSeconds))
		b.WriteString("\n")
	}

	ipcConfig := b.String()

	if err := dev.IpcSet(ipcConfig); err != nil {
		dev.Close()
		return fmt.Errorf("[wireguard] IpcSet failed for proxy '%s': %w", name, err)
	}

	if err := dev.Up(); err != nil {
		dev.Close()
		return fmt.Errorf("[wireguard] dev.Up failed for proxy '%s': %w", name, err)
	}

	// SOCKS5-Server auf lokalem Port wie üblich
	bind := cfg.Proxies.Bind
	localAddr := fmt.Sprintf("%s:%d", bind, p.Port)

	logging.Log.Infof(
		"[wireguard] Launching SOCKS proxy '%s' on %s via wireguard-go netstack",
		name, localAddr,
	)

	// Dialer: nutzt direkt tnet.DialContext (Userspace-Netstack, kein systemweites Routing)
	server := socks5.NewServer(
		socks5.WithDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
			if connectTimeoutSeconds > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(connectTimeoutSeconds)*time.Second)
				defer cancel()
			}
			return tnet.DialContext(ctx, network, addr)
		}),
		socks5.WithResolver(NewTunnelResolver(tnet)),
	)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		dev.Close()
		return fmt.Errorf("[wireguard] failed to bind listener for '%s' on %s: %w", name, localAddr, err)
	}

	inst := &wireguardInstance{
		listener: listener,
		dev:      dev,
	}

	w.mu.Lock()
	if _, exists := w.instances[name]; exists {
		w.mu.Unlock()
		_ = listener.Close()
		dev.Close()
		return fmt.Errorf("[wireguard] proxy '%s' already running", name)
	}
	w.instances[name] = inst
	w.stopFlags[name] = false
	exitCb := w.exitCallback
	w.mu.Unlock()

	// SOCKS-Serving im Hintergrund
	go func() {
		if err := server.Serve(listener); err != nil {
			logging.Log.Errorf("[wireguard] '%s' socks serve error: %v", name, err)
		}
		logging.Log.Infof("[wireguard] Proxy '%s' socks server exited", name)

		// Instance aus Registry entfernen und Device schließen
		w.mu.Lock()
		stoppedByUser := w.stopFlags[name]
		delete(w.instances, name)
		delete(w.stopFlags, name)
		w.mu.Unlock()

		inst.Stop()

		if !stoppedByUser && exitCb != nil {
			exitCb(name)
		}
	}()

	return nil
}

// Stop signals the SOCKS listener and WireGuard device to stop for the given proxy.
func (w *wireguardBackend) Stop(name string) error {
	w.mu.Lock()
	inst, ok := w.instances[name]
	if !ok || inst == nil {
		w.mu.Unlock()
		logging.Log.Infof("[wireguard] Stop called for '%s' but no active instance found", name)
		return nil
	}
	w.stopFlags[name] = true
	w.mu.Unlock()

	logging.Log.Infof("[wireguard] Stopping proxy '%s'", name)
	inst.Stop()
	logging.Log.Infof("[wireguard] Proxy '%s' stop signal sent", name)
	return nil
}

// Status returns the listener port for the running proxy, if any.
func (w *wireguardBackend) Status(name string) (int, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	inst, ok := w.instances[name]
	if !ok || inst == nil || inst.listener == nil {
		return 0, false
	}
	if tcpAddr, ok := inst.listener.Addr().(*net.TCPAddr); ok {
		return tcpAddr.Port, true
	}
	return 0, true
}

// GetInstance returns a RunningInstance for the proxy, if active.
func (w *wireguardBackend) GetInstance(name string) interfaces.RunningInstance {
	w.mu.Lock()
	defer w.mu.Unlock()

	inst, ok := w.instances[name]
	if !ok || inst == nil {
		return nil
	}
	return inst
}

// parseWGAddresses parses a WG "Address" string (e.g. "10.135.114.178/32,fd7d:.../128")
// into a slice of netip.Addr for netstack.CreateNetTUN.
func parseWGAddresses(addressField string) ([]netip.Addr, error) {
	addressField = strings.TrimSpace(addressField)
	if addressField == "" {
		return nil, fmt.Errorf("empty wg address")
	}

	parts := strings.Split(addressField, ",")
	var addrs []netip.Addr
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// IPv4/IPv6 mit Prefix
		if strings.Contains(p, "/") {
			pfx, err := netip.ParsePrefix(p)
			if err != nil {
				return nil, fmt.Errorf("invalid prefix '%s': %w", p, err)
			}
			addrs = append(addrs, pfx.Addr())
			continue
		}
		// Fallback ohne Prefix
		addr, err := netip.ParseAddr(p)
		if err != nil {
			return nil, fmt.Errorf("invalid address '%s': %w", p, err)
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no valid addresses in '%s'", addressField)
	}
	return addrs, nil
}

// parseDNSAddresses attempts to parse DNS strings into netip.Addr slice.
// Invalid entries are skipped; an empty result is allowed.
func parseDNSAddresses(dns []string) []netip.Addr {
	var out []netip.Addr
	for _, raw := range dns {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		addr, err := netip.ParseAddr(s)
		if err != nil {
			continue
		}
		out = append(out, addr)
	}
	return out
}

// optionalString returns the value for key if it is a string, otherwise
// the empty string. Missing keys are treated as empty.
func optionalString(cfg map[string]any, key string) string {
	raw, ok := cfg[key]
	if !ok || raw == nil {
		return ""
	}
	if s, ok := raw.(string); ok {
		return s
	}
	return fmt.Sprint(raw)
}

// requireString ensures a non-empty string is present for key.
func requireString(cfg map[string]any, key string) (string, error) {
	v := optionalString(cfg, key)
	if strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("missing required string config '%s'", key)
	}
	return v, nil
}

// optionalInt retrieves an integer configuration value. If the key is
// missing, the provided default is returned. Strings are parsed with
// strconv.Atoi. Any other type results in an error.
func optionalInt(cfg map[string]any, key string, def int) (int, error) {
	raw, ok := cfg[key]
	if !ok || raw == nil {
		return def, nil
	}

	switch v := raw.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid integer for '%s': %v", key, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("invalid type for '%s': %T", key, raw)
	}
}

// optionalStringSlice normalises a config value into a []string.
// Accepted forms:
//   - []string
//   - []any with string-able entries
//   - single string (will be split by comma)
func optionalStringSlice(cfg map[string]any, key string) ([]string, error) {
	raw, ok := cfg[key]
	if !ok || raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case []string:
		return v, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out, nil
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid type for '%s': %T", key, raw)
	}
}

func wgKeyBase64ToHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
