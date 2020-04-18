// +build windows

package app

import (
	"errors"
)

// DevNullFilePath is the equivalent of /dev/null.
//
// This will be /dev/null for darwin and linux.
// This will be nul for windows.
const DevNullFilePath = "nul"

// HomeDirPath returns the home directory path.
//
// This will be $HOME for darwin and linux.
// This will be %USERPROFILE% for windows.
func HomeDirPath(envContainer EnvContainer) (string, error) {
	if value := envContainer.Env("USERPROFILE"); value != "" {
		return value, nil
	}
	return "", errors.New("%USERPROFILE% is not set")
}

// CacheDirPath returns the cache directory path.
//
// This will be $XDG_CACHE_HOME for darwin and linux, falling back to $HOME/.cache.
// This will be %LocalAppData% for windows.
//
// Users cannot assume that CacheDirPath, ConfigDirPath, and DataDirPath are unique.
func CacheDirPath(envContainer EnvContainer) (string, error) {
	if value := envContainer.Env("LocalAppData"); value != "" {
		return value, nil
	}
	return "", errors.New("%LocalAppData% is not set")
}

// ConfigDirPath returns the config directory path.
//
// This will be $XDG_CONFIG_HOME for darwin and linux, falling back to $HOME/.config.
// This will be %AppData% for windows.
//
// Users cannot assume that CacheDirPath, ConfigDirPath, and DataDirPath are unique.
func ConfigDirPath(envContainer EnvContainer) (string, error) {
	if value := envContainer.Env("AppData"); value != "" {
		return value, nil
	}
	return "", errors.New("%AppData% is not set")
}

// DataDirPath returns the data directory path.
//
// This will be $XDG_DATA_HOME for darwin and linux, falling back to $HOME/.local/share.
// This will be %LocalAppData% for windows.
//
// Users cannot assume that CacheDirPath, ConfigDirPath, and DataDirPath are unique.
func DataDirPath(envContainer EnvContainer) (string, error) {
	if value := envContainer.Env("LocalAppData"); value != "" {
		return value, nil
	}
	return "", errors.New("%LocalAppData% is not set")
}
