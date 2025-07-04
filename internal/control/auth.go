package control

import "github.com/mfulz/portgeist/internal/config"

func IsControlAllowed(proxyCfg config.Proxy, user string, skip bool) bool {
	if skip {
		return true
	}
	for _, u := range proxyCfg.AllowedControls {
		if u == user {
			return true
		}
	}
	return false
}
