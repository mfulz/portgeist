package control

import (
	"encoding/json"

	"github.com/mfulz/portgeist/internal/acl"
	"github.com/mfulz/portgeist/internal/configd"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/mfulz/portgeist/internal/proxy"
	"github.com/mfulz/portgeist/protocol"
)

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

func StartProxyHandler(cfg *configd.Config, instance configd.ControlInstance) func(req *protocol.Request) *protocol.Response {
	return func(req *protocol.Request) *protocol.Response {
		var payload protocol.StartRequest
		_ = decodePayload(req.Data, &payload)

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := extractUser(req)
		if !acl.Can(user, "proxy_list", proxyCfg.ACLs) {
			return &protocol.Response{Status: "error", Error: "not allowed"}
		}

		if err := proxy.StartProxy(payload.Name, proxyCfg, cfg); err != nil {
			return &protocol.Response{Status: "error", Error: err.Error()}
		}
		return &protocol.Response{Status: "ok"}
	}
}

func StopProxyHandler(cfg *configd.Config, instance configd.ControlInstance) func(req *protocol.Request) *protocol.Response {
	return func(req *protocol.Request) *protocol.Response {
		var payload protocol.StopRequest
		_ = decodePayload(req.Data, &payload)

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := extractUser(req)
		if !acl.Can(user, "proxy_list", proxyCfg.ACLs) {
			return &protocol.Response{Status: "error", Error: "not allowed"}
		}

		if err := proxy.StopProxy(payload.Name, proxyCfg, cfg); err != nil {
			return &protocol.Response{Status: "error", Error: err.Error()}
		}
		return &protocol.Response{Status: "ok"}
	}
}

func ProxyStatusHandler(cfg *configd.Config, instance configd.ControlInstance) func(req *protocol.Request) *protocol.Response {
	return func(req *protocol.Request) *protocol.Response {
		var payload protocol.StatusRequest
		_ = decodePayload(req.Data, &payload)

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := extractUser(req)
		if !acl.Can(user, "proxy_list", proxyCfg.ACLs) {
			return &protocol.Response{Status: "error", Error: "not allowed"}
		}

		status, err := proxy.GetProxyStatus(payload.Name, proxyCfg, cfg)
		if err != nil {
			return &protocol.Response{Status: "error", Error: err.Error()}
		}
		return &protocol.Response{Status: "ok", Data: status}
	}
}

func ProxyListHandler(cfg *configd.Config, instance configd.ControlInstance) func(req *protocol.Request) *protocol.Response {
	return func(req *protocol.Request) *protocol.Response {
		user := extractUser(req)
		if !acl.Can(user, "proxy_list", acl.ACLRuleSet{}) {
			return &protocol.Response{Status: "error", Error: "not allowed"}
		}

		var result []string
		// for name, proxyCfg := range cfg.Proxies.Proxies {
		for name := range cfg.Proxies.Proxies {
			// if IsControlAllowed(proxyCfg, user, !instance.Auth.Enabled) {
			result = append(result, name)
			// }
		}
		return &protocol.Response{
			Status: "ok",
			Data: protocol.ListResponse{
				Proxies: result,
			},
		}
	}
}

func ProxyInfoHandler(cfg *configd.Config, instance configd.ControlInstance) func(req *protocol.Request) *protocol.Response {
	return func(req *protocol.Request) *protocol.Response {
		var payload protocol.InfoRequest
		_ = decodePayload(req.Data, &payload)

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := extractUser(req)
		logging.Log.Debugf("extracted user: %v", user)

		if !acl.Can(user, "proxy_info", proxyCfg.ACLs) {
			return &protocol.Response{Status: "error", Error: "not allowed"}
		}

		info, err := proxy.GetProxyInfo(payload.Name, proxyCfg, cfg)
		if err != nil {
			return &protocol.Response{Status: "error", Error: err.Error()}
		}
		return &protocol.Response{Status: "ok", Data: info}
	}
}

func ProxySetActiveHandler(cfg *configd.Config, instance configd.ControlInstance) func(req *protocol.Request) *protocol.Response {
	return func(req *protocol.Request) *protocol.Response {
		var payload protocol.SetActiveRequest
		_ = decodePayload(req.Data, &payload)

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Name]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := extractUser(req)
		if !acl.Can(user, "proxy_setactive", proxyCfg.ACLs) {
			return &protocol.Response{Status: "error", Error: "not allowed"}
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
	}
}

func ResolveProxyHandler(cfg *configd.Config, instance configd.ControlInstance) func(req *protocol.Request) *protocol.Response {
	return func(req *protocol.Request) *protocol.Response {
		var payload protocol.ResolvRequest
		_ = decodePayload(req.Data, &payload)

		proxyCfg, ok := cfg.Proxies.Proxies[payload.Alias]
		if !ok {
			return &protocol.Response{Status: "error", Error: "unknown proxy"}
		}

		user := extractUser(req)
		if !acl.Can(user, "proxy_resolve", proxyCfg.ACLs) {
			return &protocol.Response{Status: "error", Error: "not allowed"}
		}

		return &protocol.Response{
			Status: "ok",
			Data: protocol.ResolvResponse{
				Host: cfg.Proxies.Bind,
				Port: proxyCfg.Port,
			},
		}
	}
}
