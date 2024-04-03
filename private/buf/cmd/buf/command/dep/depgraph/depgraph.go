// Copyright 2020-2024 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package depgraph

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName     = "error-format"
	disableSymlinksFlagName = "disable-symlinks"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Print the dependency graph in DOT format",
		Long: `As an example, if module in directory "src/proto" depends on module "buf.build/foo/bar"
from the BSR with commit "12345", and "buf.build/foo/bar:12345" depends on module "buf.build/foo/baz"
from the BSR with commit "67890", the following will be printed:

digraph {

  "src/proto" -> "buf.build/foo/bar:12345"
  "buf.build/foo/bar:12345" -> "buf.build/foo/baz:67890"

}

The actual output may vary between CLI versions and has no stability guarantees, however the output
will always be in valid DOT format. If you'd like us to produce an alternative stable format
(such as a Protobuf message that we serialize to JSON), let us know!

See https://graphviz.org to explore Graphviz and the DOT language.
Installation of graphviz will vary by platform, but is easy to install using homebrew:

brew install graphviz

You can easily visualize a dependency graph using the dot tool:

buf dep graph | dot -Tpng >| graph.png && open graph.png

` + bufcli.GetSourceOrModuleLong(`the source or module to print the dependency graph for`),
		Args: appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	ErrorFormat     string
	DisableSymlinks bool
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
		bufctl.WithFileAnnotationErrorFormat(flags.ErrorFormat),
	)
	if err != nil {
		return err
	}
	workspace, err := controller.GetWorkspace(ctx, input)
	if err != nil {
		return err
	}
	graph, err := bufmodule.ModuleSetToDAG(workspace)
	if err != nil {
		return err
	}
	dotString, err := graph.DOTString(moduleToString)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(container.Stdout(), dotString)
	return err
}

func moduleToString(module bufmodule.Module) string {
	if moduleFullName := module.ModuleFullName(); moduleFullName != nil {
		if commitID := module.CommitID(); !commitID.IsNil() {
			return moduleFullName.String() + ":" + uuidutil.ToDashless(commitID)
		}
		return moduleFullName.String()
	}
	return module.OpaqueID()
}
