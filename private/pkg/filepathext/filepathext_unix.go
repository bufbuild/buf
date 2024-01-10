//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package filepathext

import (
	"os"
)

var (
	FSRoot = string(os.PathSeparator)
)
