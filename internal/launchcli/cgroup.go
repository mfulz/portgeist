// Package launchcli provides dynamic configuration and helpers
// for launching external processes via proxy wrappers (e.g. proxychains, torsocks).
package launchcli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteRedsocksConf creates a temporary redsocks config file.
func WriteRedsocksConf(localPort int, proxyHost string, proxyPort int) (string, error) {
	content := fmt.Sprintf(`
base {
        // debug: connection progress & client list on SIGUSR1
        log_debug = on;
        // info: start and end of client session
        log_info = on;
        log = stderr;
        // log = "file:/path/to/file";
        // detach from console
        daemon = on;

        // user = redsocks;
        // group = redsocks;
        // chroot = "/var/chroot";
        redirector = iptables;
        reuseport = on;
}

redsocks {
        bind = "127.0.0.1:%d";
        // listenq = 128; // SOMAXCONN equals 128 on my Linux box.
        relay = "%s:%d";
        type = socks5;
        autoproxy = 0;
        timeout = 10;
}
`, localPort, proxyHost, proxyPort)

	confPath := filepath.Join(os.TempDir(), fmt.Sprintf("redsocks_%d.conf", time.Now().UnixNano()))
	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return "", err
	}
	return confPath, nil
}
