package buf

import "github.com/bufbuild/cli/clicobra"

const version = "0.4.0-dev"

// Main is the main.
func Main(use string) {
	clicobra.Main(newRootCommand(use), version)
}
