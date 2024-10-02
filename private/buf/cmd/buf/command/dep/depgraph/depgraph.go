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
	"encoding/json"
	"fmt"
	"slices"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName     = "error-format"
	disableSymlinksFlagName = "disable-symlinks"
	formatFlagName          = "format"

	dotFormatString  = "dot"
	jsonFormatString = "json"
)

var (
	allGraphFormatStrings = []string{
		dotFormatString,
		jsonFormatString,
	}
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Print the dependency graph",
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
	Format       string
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
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		dotFormatString,
		fmt.Sprintf(
			"The format to print graph as. Must be one of %s",
			stringutil.SliceToString(allGraphFormatStrings),
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
	var graphString string
	switch flags.Format {
	case dotFormatString:
		dotString, err := graph.DOTString(moduleToString)
		if err != nil {
			return err
		}
		graphString = dotString
	case jsonFormatString:
		// We traverse each module (node) in the graph and populate the deps (outbound nodes).
		// We keep track of every module we have seen so we can update their d
		moduleFullNameOrOpaqueIDToExternalModule := make(map[string]externalModule)
		if err := graph.WalkNodes(
			func(module bufmodule.Module, _ []bufmodule.Module, deps []bufmodule.Module) error {
				moduleFullNameOrOpaqueID := moduleFullNameOrOpaqueID(module)
				// We have already populated this node through deps, we can skip module.
				if _, ok := moduleFullNameOrOpaqueIDToExternalModule[moduleFullNameOrOpaqueID]; ok {
					return nil
				}
				// We first scaffold a module with no deps populated yet.
				externalModule, err := externalModuleNoDepsForModule(module)
				if err != nil {
					return err
				}
				if err := externalModule.addDeps(deps, graph, moduleFullNameOrOpaqueIDToExternalModule, flags); err != nil {
					return err
				}
				// Sort the deps alphabetically before adding our external module.
				sortExternalModules(externalModule.Deps)
				moduleFullNameOrOpaqueIDToExternalModule[moduleFullNameOrOpaqueID] = externalModule
				return nil
			},
		); err != nil {
			return err
		}
		externalModules := slicesext.MapValuesToSlice(moduleFullNameOrOpaqueIDToExternalModule)
		// Sort all modules alphabetically.
		sortExternalModules(externalModules)
		data, err := json.Marshal(externalModules)
		if err != nil {
			return err
		}
		graphString = string(data)
	default:
		return appcmd.NewInvalidArgumentErrorf("invalid value for --%s: %s", formatFlagName, flags.Format)
	}
	_, err = fmt.Fprintln(container.Stdout(), graphString)
	return err
}

func moduleToString(module bufmodule.Module) string {
	if moduleFullName := module.ModuleFullName(); moduleFullName != nil {
		commitID := dashlessCommitIDStringForModule(module)
		if commitID != "" {
			return moduleFullName.String() + ":" + commitID
		}
		return moduleFullName.String()
	}
	return module.OpaqueID()
}

// moduleFullNameOrOpaqueID returns the ModuleFullName for a module if available, otherwise
// it returns the OpaqueID.
func moduleFullNameOrOpaqueID(module bufmodule.Module) string {
	if moduleFullName := module.ModuleFullName(); moduleFullName != nil {
		return moduleFullName.String()
	}
	return module.OpaqueID()
}

// dashlessCommitIDStringForModule returns the dashless UUID for the commit. If no commit
// is set, we return an empty string.
func dashlessCommitIDStringForModule(module bufmodule.Module) string {
	if commitID := module.CommitID(); commitID != uuid.Nil {
		return uuidutil.ToDashless(commitID)
	}
	return ""
}

type externalModule struct {
	// ModuleFullName if remote, OpaqueID if no ModuleFullName
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Dashless
	Commit string           `json:"commit,omitempty" yaml:"commit,omitempty"`
	Digest string           `json:"digest,omitempty" yaml:"digest,omitempty"`
	Deps   []externalModule `json:"deps,omitempty" yaml:"deps,omitempty"`
	Local  bool             `json:"local,omitempty" yaml:"local,omitempty"`
}

func (e *externalModule) addDeps(
	deps []bufmodule.Module,
	graph *dag.Graph[string, bufmodule.Module],
	moduleFullNameOrOpaqueIDToExternalModule map[string]externalModule,
	flags *flags,
) error {
	for _, dep := range deps {
		depModuleFullNameOrOpaqueID := moduleFullNameOrOpaqueID(dep)
		depExternalModule, ok := moduleFullNameOrOpaqueIDToExternalModule[depModuleFullNameOrOpaqueID]
		if ok {
			// If this dependency has already been seen, we can simply update our current module
			// and return early.
			e.Deps = append(e.Deps, depExternalModule)
			return nil
		}
		// Otherwise, we create a new external module for our direct dependency. However, we do
		// not add it to our map yet, we only add it once all transitive dependencies have been
		// handled.
		depExternalModule, err := externalModuleNoDepsForModule(dep)
		if err != nil {
			return err
		}
		transitiveDeps, err := graph.OutboundNodes(dep.OpaqueID())
		if err != nil {
			return err
		}
		if err := depExternalModule.addDeps(transitiveDeps, graph, moduleFullNameOrOpaqueIDToExternalModule, flags); err != nil {
			return err
		}
		moduleFullNameOrOpaqueIDToExternalModule[depModuleFullNameOrOpaqueID] = depExternalModule
		e.Deps = append(e.Deps, depExternalModule)
	}
	return nil
}

// externalModuleNoDepsForModule returns an externalModule for the given bufmodule.Module
// without populating the deps. This is because we want to populate the deps from the graph,
// so we handle it outside of this function.
func externalModuleNoDepsForModule(module bufmodule.Module) (externalModule, error) {
	// We always calculate the b5 digest here, we do not check the digest type that is stored
	// in buf.lock.
	digest, err := module.Digest(bufmodule.DigestTypeB5)
	if err != nil {
		return externalModule{}, err
	}
	return externalModule{
		Name:   moduleFullNameOrOpaqueID(module),
		Commit: dashlessCommitIDStringForModule(module),
		Digest: digest.String(),
		Local:  module.IsLocal(),
	}, nil
}

func sortExternalModules(externalModules []externalModule) {
	slices.SortFunc(
		externalModules,
		func(a externalModule, b externalModule) int {
			if a.Name > b.Name {
				return 1
			}
			return -1
		},
	)
}
