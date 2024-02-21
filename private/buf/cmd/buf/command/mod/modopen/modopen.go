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

package modopen

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/pkg/browser"
)

const deprecationMessage = "this command is not supported for v2 buf.yaml files as v2 buf.yaml files contain multiple modules. However, this command will continue to work for v1 buf.yaml files."

// NewCommand returns a new open Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Open the module's homepage in a web browser",
		Long: `The first argument is the directory with the buf.yaml of the module to open.

The directory must have a buf.yaml that contains a single specified module name.

The directory defaults to "." if no argument is specified.`,
		Deprecated: deprecationMessage,
		Hidden:     true,
		Args:       appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container)
			},
		),
	}
}

func run(
	ctx context.Context,
	container appext.Container,
) error {
	dirPath := "."
	if container.NumArgs() > 0 {
		dirPath = container.Arg(0)
	}
	bufYAMLFile, err := bufcli.GetBufYAMLFileForDirPath(ctx, dirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("no buf.yaml discovered in directory %s", dirPath)
		}
		return err
	}
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		moduleConfigs := bufYAMLFile.ModuleConfigs()
		if len(moduleConfigs) != 1 {
			return syserror.Newf("got %d ModuleConfigs for a v1beta1/v1 buf.yaml", len(moduleConfigs))
		}
		moduleFullName := moduleConfigs[0].ModuleFullName()
		if moduleFullName == nil {
			return fmt.Errorf("%s/buf.yaml has no module name", dirPath)
		}
		return browser.OpenURL("https://" + moduleFullName.String())
	case bufconfig.FileVersionV2:
		return errors.New(deprecationMessage)
	default:
		return syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}
