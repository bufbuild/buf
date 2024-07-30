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

package configlsmodules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	configFlagName = "config"
	formatFlagName = "format"

	formatPath = "path"
	formatName = "name"
	formatJSON = "json"

	defaultFormat = formatPath
)

var (
	allFormats = []string{
		formatPath,
		formatName,
		formatJSON,
	}
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "List configured modules",
		Args:  appcmd.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Config string
	Format string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml file or data to use for configuration.`,
	)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		defaultFormat,
		fmt.Sprintf(
			"The format to print rules as. Must be one of %s",
			stringutil.SliceToString(allFormats),
		),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	externalModules, err := getExternalModules(ctx, flags.Config)
	if err != nil {
		return err
	}
	return printExternalModules(ctx, container, flags.Format, externalModules)
}

func getExternalModules(
	ctx context.Context,
	configOverride string,
) ([]*externalModule, error) {
	// If an override is specified, read buf.yaml from it.
	if configOverride != "" {
		bufYAMLFile, err := bufconfig.GetBufYAMLFileForOverride(configOverride)
		if err != nil {
			return nil, err
		}
		return getExternalModulesForBufYAMLFile(ctx, bufYAMLFile)
	}
	// First, look for a buf.work.yaml file.
	bufWorkYAMLFile, err := bufcli.GetBufWorkYAMLFileForDirPath(ctx, ".")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		// We do not have a buf.work.yaml file, attempt to read a buf.yaml file.
		bufYAMLFile, err := bufcli.GetBufYAMLFileForDirPath(ctx, ".")
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
			// We do not have a buf.work.yaml or buf.yaml file, use the default.
			bufYAMLFile, err = bufconfig.NewBufYAMLFile(
				bufconfig.FileVersionV2,
				[]bufconfig.ModuleConfig{
					bufconfig.DefaultModuleConfigV2,
				},
				nil,
			)
			if err != nil {
				return nil, err
			}
		}
		// This handles both buf.yaml file and no file courtesy of the above logic.
		return getExternalModulesForBufYAMLFile(ctx, bufYAMLFile)
	}
	// We did have a buf.work.yaml file, but before handling it, check there is not a buf.yaml.
	_, err = bufcli.GetBufYAMLFileForDirPath(ctx, ".")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		return nil, errors.New("Both buf.work.yaml and buf.yaml found. It is not valid to have a buf.work.yaml and buf.yaml in the same directory, buf.work.yaml specifies a workspace of modules, while buf.yaml either specifies a single module or a workspace of modules itself.")
	}
	// Handle the buf.work.yaml.
	return getExternalModulesForBufWorkYAMLFile(ctx, bufWorkYAMLFile)
}

func getExternalModulesForBufWorkYAMLFile(
	ctx context.Context,
	bufWorkYAMLFile bufconfig.BufWorkYAMLFile,
) ([]*externalModule, error) {
	var externalModules []*externalModule
	for _, dirPath := range bufWorkYAMLFile.DirPaths() {
		bufYAMLFile, err := bufcli.GetBufYAMLFileForDirPath(ctx, dirPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
			externalModules = append(
				externalModules,
				newExternalModule(dirPath, ""),
			)
			continue
		}
		// This is a sanity check. Make sure we have what we expect.
		switch bufYAMLFile.FileVersion() {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			moduleConfigs := bufYAMLFile.ModuleConfigs()
			if len(moduleConfigs) != 1 {
				return nil, syserror.Newf("got BufYAMLFile at %q with FileVersion %v with %d ModuleConfigs", dirPath, bufYAMLFile.FileVersion(), len(moduleConfigs))
			}
			moduleConfig := moduleConfigs[0]
			if moduleConfig.DirPath() != "." {
				return nil, syserror.Newf("got BufYAMLFile at %q with FileVersion %v with ModuleConfig that had non-root DirPath %q", dirPath, bufYAMLFile.FileVersion(), moduleConfig.DirPath())
			}
			var name string
			if moduleFullName := moduleConfig.ModuleFullName(); moduleFullName != nil {
				name = moduleFullName.String()
			}
			externalModules = append(
				externalModules,
				// The dirPath is the path specified in the buf.work.yaml.
				// The DirPath for v1/v1beta1 ModuleConfigs is always ".".
				newExternalModule(dirPath, name),
			)
		case bufconfig.FileVersionV2:
			return nil, fmt.Errorf("buf.work.yaml pointed to directory %q which has a v2 buf.yaml file", dirPath)
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", bufYAMLFile.FileVersion())
		}
	}
	return externalModules, nil
}

func getExternalModulesForBufYAMLFile(
	ctx context.Context,
	bufYAMLFile bufconfig.BufYAMLFile,
) ([]*externalModule, error) {
	moduleConfigs := bufYAMLFile.ModuleConfigs()
	externalModules := make([]*externalModule, len(moduleConfigs))
	for i, moduleConfig := range moduleConfigs {
		var name string
		if moduleFullName := moduleConfig.ModuleFullName(); moduleFullName != nil {
			name = moduleFullName.String()
		}
		externalModules[i] = newExternalModule(moduleConfig.DirPath(), name)
	}
	return externalModules, nil
}

func printExternalModules(
	ctx context.Context,
	container app.StdoutContainer,
	format string,
	externalModules []*externalModule,
) error {
	switch format {
	case formatPath:
		sort.Slice(
			externalModules,
			func(i int, j int) bool {
				return externalModules[i].Path < externalModules[j].Path
			},
		)
		for _, externalModule := range externalModules {
			if _, err := container.Stdout().Write([]byte(externalModule.Path + "\n")); err != nil {
				return err
			}
		}
		return nil
	case formatName:
		sort.Slice(
			externalModules,
			func(i int, j int) bool {
				return externalModules[i].Name < externalModules[j].Name
			},
		)
		for _, externalModule := range externalModules {
			if externalModule.Name == "" {
				continue
			}
			if _, err := container.Stdout().Write([]byte(externalModule.Name + "\n")); err != nil {
				return err
			}
		}
		return nil
	case formatJSON:
		sort.Slice(
			externalModules,
			func(i int, j int) bool {
				return externalModules[i].Path < externalModules[j].Path
			},
		)
		for _, externalModule := range externalModules {
			data, err := json.Marshal(externalModule)
			if err != nil {
				return err
			}
			if _, err := container.Stdout().Write([]byte(string(data) + "\n")); err != nil {
				return err
			}
		}
		return nil
	default:
		return appcmd.NewInvalidArgumentErrorf("unknown value for --%s: %s", formatFlagName, format)
	}
}

type externalModule struct {
	Path string `json:"path" yaml:"path"`
	Name string `json:"name" yaml:"name"`
}

func newExternalModule(path string, name string) *externalModule {
	return &externalModule{
		Path: path,
		Name: name,
	}
}
