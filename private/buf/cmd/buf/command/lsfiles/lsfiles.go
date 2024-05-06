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

package lsfiles

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/spf13/pflag"
)

const (
	formatFlagName            = "format"
	configFlagName            = "config"
	errorFormatFlagName       = "error-format"
	includeImportsFlagName    = "include-imports"
	includeImportableFlagName = "include-importable"
	pathsFlagName             = "path"
	excludePathsFlagName      = "exclude-path"
	disableSymlinksFlagName   = "disable-symlinks"
	asImportPathsFlagName     = "as-import-paths"

	formatText   = "text"
	formatJSON   = "json"
	formatImport = "import"
)

var (
	allFormats = []string{formatText, formatJSON, formatImport}
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "List Protobuf files",
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
	Format            string
	Config            string
	IncludeImports    bool
	IncludeImportable bool
	Paths             []string
	ExcludePaths      []string
	DisableSymlinks   bool
	// Deprecated. This flag no longer has any effect as we don't build images anymore.
	ErrorFormat string
	// Deprecated
	AsImportPaths bool
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindPaths(flagSet, &f.Paths, pathsFlagName)
	bufcli.BindExcludePaths(flagSet, &f.ExcludePaths, excludePathsFlagName)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml file or data to use for configuration`,
	)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		formatText,
		fmt.Sprintf(
			`The format to print the Protofile files. Must be one of %s`,
			strings.Join(allFormats, ", "),
		),
	)
	flagSet.BoolVar(
		&f.IncludeImports,
		includeImportsFlagName,
		false,
		"Include imports",
	)
	flagSet.BoolVar(
		&f.IncludeImportable,
		includeImportableFlagName,
		false,
		fmt.Sprintf(
			"Include all .proto file that are importable by the input. --%s is redundant if this is set",
			includeImportsFlagName,
		),
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	_ = flagSet.MarkDeprecated(errorFormatFlagName, "This flag no longer has any effect")
	_ = flagSet.MarkHidden(errorFormatFlagName)
	flagSet.BoolVar(
		&f.AsImportPaths,
		asImportPathsFlagName,
		false,
		"Strip local directory paths and print filepaths as they are imported",
	)
	_ = flagSet.MarkDeprecated(asImportPathsFlagName, fmt.Sprintf("Use --%s=import instead", formatFlagName))
	_ = flagSet.MarkHidden(asImportPathsFlagName)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	if flags.AsImportPaths {
		flags.Format = formatImport
	}
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
	// Sorted by Path.
	imageFileInfos, err := controller.GetImportableImageFileInfos(
		ctx,
		input,
		bufctl.WithTargetPaths(flags.Paths, flags.ExcludePaths),
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
	}
	if !flags.IncludeImportable {
		if !flags.IncludeImports {
			imageFileInfos = slicesext.Filter(
				imageFileInfos,
				func(imageFileInfo bufimage.ImageFileInfo) bool {
					return !imageFileInfo.IsImport()
				},
			)
		} else {
			// Also automatically adds imported WKTs if not present.
			imageFileInfos, err = bufimage.ImageFileInfosWithOnlyTargetsAndTargetImports(
				ctx,
				datawkt.ReadBucket,
				imageFileInfos,
			)
			if err != nil {
				return err
			}
		}
	}
	var formatFunc func(bufimage.ImageFileInfo) (string, error)
	switch flags.Format {
	case formatText:
		sort.Slice(
			imageFileInfos,
			func(i int, j int) bool {
				return imageFileInfos[i].ExternalPath() < imageFileInfos[j].ExternalPath()
			},
		)
		formatFunc = func(imageFileInfo bufimage.ImageFileInfo) (string, error) {
			return imageFileInfo.ExternalPath(), nil
		}
	case formatJSON:
		sort.Slice(
			imageFileInfos,
			func(i int, j int) bool {
				if imageFileInfos[i].LocalPath() < imageFileInfos[j].LocalPath() {
					return true
				}
				if imageFileInfos[i].LocalPath() > imageFileInfos[j].LocalPath() {
					return false
				}
				return imageFileInfos[i].Path() < imageFileInfos[j].Path()
			},
		)
		formatFunc = func(imageFileInfo bufimage.ImageFileInfo) (string, error) {
			data, err := json.Marshal(newExternalImageFileInfo(imageFileInfo))
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	case formatImport:
		formatFunc = func(imageFileInfo bufimage.ImageFileInfo) (string, error) {
			return imageFileInfo.Path(), nil
		}
	default:
		return appcmd.NewInvalidArgumentErrorf("--%s must be one of %s", formatFlagName, strings.Join(allFormats, ", "))
	}
	lines, err := slicesext.MapError(imageFileInfos, formatFunc)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(container.Stdout(), line); err != nil {
			return err
		}
	}
	return nil
}

type externalImageFileInfo struct {
	Path       string `json:"path" yaml:"path"`
	ImportPath string `json:"import_path" yaml:"import_path"`
	Module     string `json:"module" yaml:"module"`
	// Dashless
	Commit   string `json:"commit" yaml:"commit"`
	IsImport bool   `json:"is_import" yaml:"is_import"`
}

func newExternalImageFileInfo(imageFileInfo bufimage.ImageFileInfo) *externalImageFileInfo {
	var module string
	if moduleFullName := imageFileInfo.ModuleFullName(); moduleFullName != nil {
		module = moduleFullName.String()
	}
	var commit string
	if commitID := imageFileInfo.CommitID(); !commitID.IsNil() {
		commit = uuidutil.ToDashless(commitID)
	}
	return &externalImageFileInfo{
		Path: imageFileInfo.LocalPath(),
		// This seems backwards when you read it, but it is right: the Path is the import path,
		// the LocalPath is the path that a user would have for a file on disk.
		ImportPath: imageFileInfo.Path(),
		Module:     module,
		Commit:     commit,
		IsImport:   imageFileInfo.IsImport(),
	}
}
