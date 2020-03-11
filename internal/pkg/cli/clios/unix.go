// +build darwin linux

package clios

import (
	"errors"
	"path/filepath"
)

// DevNull is the equivalent of /dev/null for darwin, linux, and windows.
//
// This will be /dev/null for darwin and linux.
// This will be nul for windows.
const DevNull = "/dev/null"

// Home outputs the equivalent of $HOME for darwin, linux, and windows.
//
// This will be $HOME for darwin and linux.
// This will be %HOMEDRIVE%%HOMEPATH% for windows, falling back to %USERPROFILE%.
func Home(getenv func(string) string) (string, error) {
	if home := getenv("HOME"); home != "" {
		return home, nil
	}
	return "", errors.New("$HOME not set")
}

// XdgConfigHome returns the equivalent of $XDG_CONFIG_HOME for darwin, linux, and windows.
//
// This is suitable for a configuration directory.
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
//
// This will be $XDG_CONFIG_HOME for darwin and linux, falling back to $HOME/.config.
// This will be %LOCALAPPDATA% for windows, falling back to $HOME/AppData/Local.
//
// Users cannot assume that XDG_CONFIG_HOME, XDG_CACHE_HOME, and XDG_DATA_HOME are unique.
func XdgConfigHome(getenv func(string) string) (string, error) {
	return xdgValue(getenv, "XDG_CONFIG_HOME", ".config")
}

// XdgCacheHome returns the equivalent of $XDG_CACHE_HOME for darwin, linux, and windows.
//
// This is suitable for a cache directory.
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
//
// This will be $XDG_CACHE_HOME for darwin and linux, falling back to $HOME/.cache.
// This will be %LOCALAPPDATA% for windows, falling back to $HOME/AppData/Local.
//
// Users cannot assume that XDG_CONFIG_HOME, XDG_CACHE_HOME, and XDG_DATA_HOME are unique.
func XdgCacheHome(getenv func(string) string) (string, error) {
	return xdgValue(getenv, "XDG_CACHE_HOME", ".cache")
}

// XdgDataHome returns the equivalent of $XDG_DATA_HOME for darwin, linux, and windows.
//
// This is suitable for a data directory.
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
//
// This will be $XDG_DATA_HOME for darwin and linux, falling back to $HOME/.local/share.
// This will be %LOCALAPPDATA% for windows, falling back to $HOME/AppData/Local.
//
// Users cannot assume that XDG_CONFIG_HOME, XDG_CACHE_HOME, and XDG_DATA_HOME are unique.
func XdgDataHome(getenv func(string) string) (string, error) {
	return xdgValue(getenv, "XDG_DATA_HOME", filepath.Join(".local", "share"))
}

func xdgValue(getenv func(string) string, key string, fallbackDirPathRelHome string) (string, error) {
	if value := getenv(key); value != "" {
		return value, nil
	}
	home, err := Home(getenv)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, fallbackDirPathRelHome), nil
}
