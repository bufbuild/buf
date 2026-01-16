// Copyright 2020-2025 Buf Technologies, Inc.
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

package edit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"buf.build/go/standard/xstrings"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/spf13/pflag"
)

const (
	configFlagName          = "config"
	deprecateFlagName       = "deprecate"
	diffFlagName            = "diff"
	diffFlagShortName       = "d"
	disableSymlinksFlagName = "disable-symlinks"
	errorFormatFlagName     = "error-format"
	excludePathsFlagName    = "exclude-path"
	pathsFlagName           = "path"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Edit Protobuf source files",
		Long: `
Edit Protobuf source files in-place.

By default, the source is the current directory and files are formatted and rewritten in-place.

Examples:

Edit all files in the current directory:

    $ buf source edit

Edit files and display a diff of the changes:

    $ buf source edit -d

Deprecate all types under a package prefix:

    $ buf source edit --deprecate foo.bar

The --deprecate flag adds the 'deprecated = true' option to all types whose
fully-qualified name starts with the given prefix. For fields and enum values,
only exact matches are deprecated (they are not included in recursive deprecation).

Multiple --deprecate flags can be specified:

    $ buf source edit --deprecate foo.bar --deprecate baz.qux

Deprecate a specific field:

    $ buf source edit --deprecate foo.bar.MyMessage.my_field
`,
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
	Config          string
	Deprecate       []string
	Diff            bool
	DisableSymlinks bool
	ErrorFormat     string
	ExcludePaths    []string
	Paths           []string
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
	flagSet.BoolVarP(
		&f.Diff,
		diffFlagName,
		diffFlagShortName,
		false,
		"Display diffs instead of rewriting files",
	)
	flagSet.StringSliceVar(
		&f.Deprecate,
		deprecateFlagName,
		nil,
		`The prefix of the types (package, message, enum, extension, service, method) to deprecate.
When specified, all types under the prefix will have the 'deprecated' option added to them.`,
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s",
			xstrings.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml file or data to use for configuration`,
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	// We use getDirOrProtoFileRef to see if we have a valid DirOrProtoFileRef.
	// This is needed to write files in-place.
	sourceDirOrProtoFileRef, sourceDirOrProtoFileRefErr := getDirOrProtoFileRef(ctx, container, source)
	if sourceDirOrProtoFileRefErr != nil {
		if errors.Is(sourceDirOrProtoFileRefErr, buffetch.ErrModuleFormatDetectedForDirOrProtoFileRef) {
			return appcmd.NewInvalidArgumentErrorf("invalid input %q: must be a directory or proto file", source)
		}
		return appcmd.NewInvalidArgumentErrorf("invalid input %q: %v", source, sourceDirOrProtoFileRefErr)
	}
	if err := validateNoIncludePackageFiles(sourceDirOrProtoFileRef); err != nil {
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
	workspace, err := controller.GetWorkspace(
		ctx,
		source,
		bufctl.WithTargetPaths(flags.Paths, flags.ExcludePaths),
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
	}
	moduleReadBucket := bufmodule.ModuleReadBucketWithOnlyTargetFiles(
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFilesForTargetModules(workspace),
	)
	originalReadBucket := bufmodule.ModuleReadBucketToStorageReadBucket(moduleReadBucket)

	// Build format options
	var formatOpts []bufformat.FormatOption
	for _, deprecatePrefix := range flags.Deprecate {
		formatOpts = append(formatOpts, bufformat.WithDeprecate(deprecatePrefix))
	}

	formattedReadBucket, err := bufformat.FormatBucket(ctx, originalReadBucket, formatOpts...)
	if err != nil {
		return err
	}

	diffBuffer := bytes.NewBuffer(nil)
	changedPaths, err := storage.DiffWithFilenames(
		ctx,
		diffBuffer,
		originalReadBucket,
		formattedReadBucket,
		storage.DiffWithExternalPaths(),
	)
	if err != nil {
		return err
	}
	diffExists := diffBuffer.Len() > 0

	if flags.Diff {
		if diffExists {
			if _, err := io.Copy(container.Stdout(), diffBuffer); err != nil {
				return err
			}
		}
		return nil
	}

	// Write files in-place (default behavior)
	changedPathSet := xslices.ToStructMap(changedPaths)
	return storage.WalkReadObjects(
		ctx,
		formattedReadBucket,
		"",
		func(readObject storage.ReadObject) error {
			if _, ok := changedPathSet[readObject.Path()]; !ok {
				// no change, nothing to re-write
				return nil
			}
			file, err := os.OpenFile(readObject.ExternalPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer func() {
				retErr = errors.Join(retErr, file.Close())
			}()
			if _, err := file.ReadFrom(readObject); err != nil {
				return err
			}
			return nil
		},
	)
}

func getDirOrProtoFileRef(
	ctx context.Context,
	container appext.Container,
	value string,
) (buffetch.DirOrProtoFileRef, error) {
	return buffetch.NewDirOrProtoFileRefParser(
		container.Logger(),
	).GetDirOrProtoFileRef(ctx, value)
}

func validateNoIncludePackageFiles(dirOrProtoFileRef buffetch.DirOrProtoFileRef) error {
	if protoFileRef, ok := dirOrProtoFileRef.(buffetch.ProtoFileRef); ok && protoFileRef.IncludePackageFiles() {
		return appcmd.NewInvalidArgumentError("cannot specify include_package_files=true with source edit")
	}
	return nil
}
