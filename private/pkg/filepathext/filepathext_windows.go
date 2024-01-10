//go:build windows
// +build windows

package filepathext

import (
	"os"
)

var (
	// The environment variable that shows the drive that holds the
	// Windows folder. This is a drive name and not a folder name (`C:` not `C:\`).
	// https://learn.microsoft.com/en-us/windows/deployment/usmt/usmt-recognized-environment-variables#variables-that-are-processed-for-the-operating-system-and-in-the-context-of-each-user
	FSRoot = os.Getenv("SYSTEMDRIVE") + string(os.PathSeparator)
)
