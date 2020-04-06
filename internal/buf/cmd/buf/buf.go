package buf

import "github.com/bufbuild/buf/internal/pkg/cli/clicobra"

const version = "0.10.0"

// Main is the main.
func Main(use string, options ...RootCommandOption) {
	clicobra.Main(newRootCommand(use, options...), version)
}

// NewRootCommand creates a new root Command.
func NewRootCommand(use string, options ...RootCommandOption) *clicobra.Command {
	return newRootCommand(use, options...)
}

// RootCommandOption is an option for a root Command.
type RootCommandOption func(*clicobra.Command, *Flags)
