package buf

import (
	"context"

	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
)

const version = "0.12.0-dev"

// Main is the main.
func Main(use string, options ...RootCommandOption) {
	appcmd.Main(context.Background(), newRootCommand(use, options...), version)
}

// RootCommandOption is an option for a root Command.
type RootCommandOption func(*appcmd.Command, appflag.Builder)
