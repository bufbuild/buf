package buf

import "github.com/bufbuild/buf/internal/pkg/cli/clicobra"

const version = "0.1.0"

var develMode = ""

// Main is the main.
func Main(use string) {
	clicobra.Main(newRootCommand(use, develMode == "1"), version)
}
