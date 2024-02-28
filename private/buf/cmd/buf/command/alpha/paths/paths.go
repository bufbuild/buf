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

package paths

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/spf13/pflag"
)

const (
	configFlagName          = "config"
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
		Short: "List all available importable paths for the input",
		Long:  bufcli.GetInputLong(`the source, module, or image to list from`),
		Args:  appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Config          string
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
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml configuration file or data to use`,
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
	)
	if err != nil {
		return err
	}
	workspace, err := controller.GetWorkspace(
		ctx,
		input,
		bufctl.WithConfigOverride(flags.Config),
	)
	fileInfos, err := bufmodule.GetFileInfos(
		ctx,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace),
	)
	externalFileInfos := slicesext.Map(
		fileInfos,
		newExternalFileInfo,
	)
	sort.Slice(
		externalFileInfos,
		func(i int, j int) bool {
			return externalFileInfos[i].Path < externalFileInfos[j].Path
		},
	)
	for _, externalFileInfo := range externalFileInfos {
		data, err := json.Marshal(externalFileInfo)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(container.Stdout(), string(data)); err != nil {
			return err
		}
	}
	return nil
}

type externalFileInfo struct {
	Path         string `json:"path" yaml:"path"`
	ExternalPath string `json:"external_path" yaml:"external_path"`
	Module       string `json:"module" yaml:"module"`
	Commit       string `json:"commit" yaml:"commit"`
	Target       bool   `json:"target" yaml:"target"`
}

func newExternalFileInfo(fileInfo bufmodule.FileInfo) *externalFileInfo {
	var module string
	if moduleFullName := fileInfo.Module().ModuleFullName(); moduleFullName != nil {
		module = moduleFullName.String()
	}
	var commit string
	if commitID := fileInfo.Module().CommitID(); !commitID.IsNil() {
		commit = commitID.String()
	}
	return &externalFileInfo{
		Path:         fileInfo.Path(),
		ExternalPath: fileInfo.ExternalPath(),
		Module:       module,
		Commit:       commit,
		Target:       fileInfo.IsTargetFile(),
	}
}
