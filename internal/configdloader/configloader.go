package configloader

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveConfigPath returns the best config path for a given subsystem and filename.
// It checks, in order:
// 1. $PORTGEIST_CONFIG if set (absolute path)
// 2. ~/.portgeist/<subsystem>/<file>
// 3. /etc/portgeist/<file>
func ResolveConfigPath(subsystem, file string) (string, error) {
	if env := os.Getenv("PORTGEIST_CONFIG"); env != "" {
		return env, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		userPath := filepath.Join(home, ".portgeist", subsystem, file)
		if _, err := os.Stat(userPath); err == nil {
			return userPath, nil
		}
	}
	systemPath := filepath.Join("/etc/portgeist", file)
	if _, err := os.Stat(systemPath); err == nil {
		return systemPath, nil
	}
	return "", fmt.Errorf("no config found for %s/%s", subsystem, file)
}
