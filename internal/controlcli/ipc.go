// Package controlcli provides shared client-side IPC wrappers for interacting with geistd.
// This module unifies command execution and abstracts the SendCommandWithAuth layer.
package controlcli

import (
	"encoding/json"
	"fmt"

	"github.com/mfulz/portgeist/internal/configcli"
	"github.com/mfulz/portgeist/internal/logging"
	"github.com/mfulz/portgeist/protocol"
)

// execWithAuth dispatches a command to a daemon using configured or overridden settings.
// It wraps SendCommandWithAuth / SendDirectCommand depending on overrideAddr.
// The result is logged and returned as *protocol.Response or nil on failure.
func execWithAuth(
	cmd string,
	payload interface{},
	proxyName string,
	cfg *configcli.Config,
	daemonName string,
	overrideAddr string,
	overrideToken string,
	controlUser string,
	successMsg string,
) (*protocol.Response, error) {
	if daemonName == "" {
		daemonName = GuessDefaultDaemon(cfg)
	}

	var resp *protocol.Response
	var err error

	if overrideAddr != "" {
		resp, err = SendDirectCommand(overrideAddr, overrideToken, controlUser, cmd, payload)
	} else {
		resp, err = SendCommandWithAuth(cfg, daemonName, controlUser, cmd, payload)
	}

	if err != nil {
		logging.Log.Errorf("Error: %v\n", err)
		return nil, err
	}
	if resp.Status != "ok" {
		logging.Log.Errorf("Error: %s\n", resp.Error)
		return resp, fmt.Errorf("%s", resp.Error)
	}
	if successMsg != "" {
		logging.Log.Infof(successMsg, proxyName)
	}
	return resp, nil
}

// StartProxy sends CmdProxyStart for the given proxy name.
func StartProxy(name string, cfg *configcli.Config, daemonName, overrideAddr, overrideToken, user string) error {
	_, err := execWithAuth(protocol.CmdProxyStart, protocol.StartRequest{Name: name}, name, cfg, daemonName, overrideAddr, overrideToken, user, "Requested start of proxy: %s\n")
	return err
}

// StopProxy sends CmdProxyStop for the given proxy name.
func StopProxy(name string, cfg *configcli.Config, daemonName, overrideAddr, overrideToken, user string) error {
	_, err := execWithAuth(protocol.CmdProxyStop, protocol.StopRequest{Name: name}, name, cfg, daemonName, overrideAddr, overrideToken, user, "Requested start of proxy: %s\n")
	return err
}

// ProxyStatus sends CmdProxyStatus for the given proxy name.
func ProxyStatus(name string, cfg *configcli.Config, daemonName, overrideAddr, overrideToken, user string) (*protocol.StatusResponse, error) {
	resp, err := execWithAuth(protocol.CmdProxyStatus, protocol.StatusRequest{Name: name}, name, cfg, daemonName, overrideAddr, overrideToken, user, "")
	if err != nil {
		return nil, err
	}
	var status protocol.StatusResponse
	data, _ := json.Marshal(resp.Data)
	if err := json.Unmarshal(data, &status); err != nil {
		logging.Log.Errorf("Failed to parse InfoResponse: %v", err)
		return nil, err
	}
	return &status, nil
}

// ProxyInfo sends CmdProxyInfo for the given proxy name.
func ProxyInfo(name string, cfg *configcli.Config, daemonName, overrideAddr, overrideToken, user string) (*protocol.InfoResponse, error) {
	resp, err := execWithAuth(protocol.CmdProxyInfo, protocol.InfoRequest{Name: name}, name, cfg, daemonName, overrideAddr, overrideToken, user, "")
	if err != nil {
		return nil, err
	}
	var info protocol.InfoResponse
	data, _ := json.Marshal(resp.Data)
	if err := json.Unmarshal(data, &info); err != nil {
		logging.Log.Errorf("Failed to parse InfoResponse: %v", err)
		return nil, err
	}
	return &info, nil
}

// SetActiveProxy sends CmdProxySetActive to change the active host for a proxy.
func SetActiveProxy(name string, cfg *configcli.Config, daemonName, overrideAddr, overrideToken, user, host string) error {
	_, err := execWithAuth(protocol.CmdProxySetActive, protocol.SetActiveRequest{Name: name, Host: host}, name, cfg, daemonName, overrideAddr, overrideToken, user, "")
	return err
}

// ProxyList sends CmdProxyList and returns a slice of proxy names.
func ProxyList(cfg *configcli.Config, daemonName, overrideAddr, overrideToken, user string) (*protocol.ListResponse, error) {
	resp, err := execWithAuth(protocol.CmdProxyList, nil, "list", cfg, daemonName, overrideAddr, overrideToken, user, "")
	if err != nil {
		return nil, err
	}
	var list protocol.ListResponse
	data, _ := json.Marshal(resp.Data)
	if err := json.Unmarshal(data, &list); err != nil {
		logging.Log.Errorf("Failed to parse ListResponse: %v", err)
		return nil, err
	}
	return &list, nil
}

// ResolveProxy resolves a proxy alias via daemon and returns host/port.
func ResolveProxy(alias string, cfg *configcli.Config, daemonName, overrideAddr, overrideToken, user string) (*protocol.ResolvResponse, error) {
	resp, err := execWithAuth(protocol.CmdProxyResolv, protocol.ResolvRequest{Alias: alias}, alias, cfg, daemonName, overrideAddr, overrideToken, user, "")
	if err != nil {
		return nil, err
	}
	var resolve protocol.ResolvResponse
	data, _ := json.Marshal(resp.Data)
	if err := json.Unmarshal(data, &resolve); err != nil {
		logging.Log.Errorf("ResolveProxy unmarshal failed: %v", err)
		return nil, err
	}
	return &resolve, nil
}
