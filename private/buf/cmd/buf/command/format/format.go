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

package format

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
)

const (
	configFlagName          = "config"
	diffFlagName            = "diff"
	diffFlagShortName       = "d"
	disableSymlinksFlagName = "disable-symlinks"
	errorFormatFlagName     = "error-format"
	excludePathsFlagName    = "exclude-path"
	exitCodeFlagName        = "exit-code"
	outputFlagName          = "output"
	outputFlagShortName     = "o"
	pathsFlagName           = "path"
	writeFlagName           = "write"
	writeFlagShortName      = "w"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Format Protobuf files",
		Long: `
By default, the source is the current directory and the formatted content is written to stdout.

Examples:

Write the current directory's formatted content to stdout:

    $ buf format

Most people will want to rewrite the files defined in the current directory in-place with -w:

    $ buf format -w

Display a diff between the original and formatted content with -d
Write a diff instead of the formatted file:

    $ buf format simple/simple.proto -d

    $ diff -u simple/simple.proto.orig simple/simple.proto
    --- simple/simple.proto.orig	2022-03-24 09:44:10.000000000 -0700
    +++ simple/simple.proto	2022-03-24 09:44:10.000000000 -0700
    @@ -2,8 +2,7 @@

    package simple;

    -
    message Object {
    -    string key = 1;
    -   bytes value = 2;
    +  string key = 1;
    +  bytes value = 2;
    }

Use the --exit-code flag to exit with a non-zero exit code if there is a diff:

    $ buf format --exit-code
    $ buf format -w --exit-code
    $ buf format -d --exit-code

Format a file, directory, or module reference by specifying a source e.g.
Write the formatted file to stdout:

    $ buf format simple/simple.proto

    syntax = "proto3";

    package simple;

    message Object {
      string key = 1;
      bytes value = 2;
    }

Write the formatted directory to stdout:

    $ buf format simple
    ...

Write the formatted module reference to stdout:

    $ buf format buf.build/acme/petapis
    ...

Write the result to a specified output file or directory with -o e.g.

Write the formatted file to another file:

    $ buf format simple/simple.proto -o simple/simple.formatted.proto

Write the formatted directory to another directory, creating it if it doesn't exist:

    $ buf format proto -o formatted

This also works with module references:

    $ buf format buf.build/acme/weather -o formatted

Rewrite the file(s) in-place with -w. e.g.

Rewrite a single file in-place:

    $ buf format simple.proto -w

Rewrite an entire directory in-place:

    $ buf format proto -w

Write a diff and rewrite the file(s) in-place:

    $ buf format simple -d -w

    $ diff -u simple/simple.proto.orig simple/simple.proto
    ...

The -w and -o flags cannot be used together in a single invocation.
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
	Diff            bool
	DisableSymlinks bool
	ErrorFormat     string
	ExcludePaths    []string
	ExitCode        bool
	Paths           []string
	Output          string
	Write           bool
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
	flagSet.BoolVar(
		&f.ExitCode,
		exitCodeFlagName,
		false,
		"Exit with a non-zero exit code if files were not already formatted",
	)
	flagSet.BoolVarP(
		&f.Write,
		writeFlagName,
		writeFlagShortName,
		false,
		"Rewrite files in-place",
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
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"-",
		fmt.Sprintf(
			`The output location for the formatted files. Must be one of format %s. If omitted, the result is written to stdout`,
			buffetch.DirOrProtoFileFormatsString,
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
	// We use getDirOrProtoFileRef to see if we have a valid DirOrProtoFileRef, and if so,
	// whether or not we have IncludePackageFiles Set.
	//
	// We abuse ExternalPaths below to say that if flags.Write is set, just write over
	// the ExternalPath, You can only really use flags.Write if you have a dir
	// or proto file. So, we abuse getDirOrProtoFileRef to determine if we have a writable source.
	// if flags.Write is set
	//
	// We also want to check that if we have a ProtoFileRef, we don't have IncludePackageFiles
	// set, regardless of if flags.Write is set.
	sourceDirOrProtoFileRef, sourceDirOrProtoFileRefErr := getDirOrProtoFileRef(ctx, container, source)
	if sourceDirOrProtoFileRefErr == nil {
		if err := validateNoIncludePackageFiles(sourceDirOrProtoFileRef); err != nil {
			return err
		}
	}
	if flags.Write {
		if flags.Output != "-" {
			return appcmd.NewInvalidArgumentErrorf("cannot use --%s when using --%s", outputFlagName, writeFlagName)
		}
		// We abuse ExternalPaths below to say that if flags.Write is set, just write over
		// the ExternalPath. Also, you can only really use flags.Write if you have a dir
		// or proto file. So, we abuse getDirOrProtoFileRef to determine if we have a writable source.
		if sourceDirOrProtoFileRefErr != nil {
			if errors.Is(sourceDirOrProtoFileRefErr, buffetch.ErrModuleFormatDetectedForDirOrProtoFileRef) {
				return appcmd.NewInvalidArgumentErrorf("invalid input %q when using --%s: must be a directory or proto file", source, writeFlagName)
			}
			return appcmd.NewInvalidArgumentErrorf("invalid input %q when using --%s: %v", source, writeFlagName, sourceDirOrProtoFileRefErr)
		}
	}
	dirOrProtoFileRef, err := getDirOrProtoFileRef(ctx, container, flags.Output)
	if err != nil {
		if errors.Is(err, buffetch.ErrModuleFormatDetectedForDirOrProtoFileRef) {
			return appcmd.NewInvalidArgumentErrorf("--%s must be a directory or proto file", outputFlagName)
		}
		return err
	}
	if err := validateNoIncludePackageFiles(dirOrProtoFileRef); err != nil {
		return err
	}

	runner := command.NewRunner()
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
		// We only want to start with the target Modules. Otherwise, we're going to fetch potential
		// ModuleDeps that are not targeted, which may result in buf format making remote calls
		// when all we care to do is format local files.
		//
		// We need to make remote Modules even lazier to make sure that buf format is really
		// not making these remote calls, but this is one component of it.
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFilesForTargetModules(workspace),
	)
	originalReadBucket := bufmodule.ModuleReadBucketToStorageReadBucket(moduleReadBucket)
	formattedReadBucket, err := bufformat.FormatBucket(ctx, originalReadBucket)
	if err != nil {
		return err
	}

	diffBuffer := bytes.NewBuffer(nil)
	changedPaths, err := storage.DiffWithFilenames(
		ctx,
		runner,
		diffBuffer,
		originalReadBucket,
		formattedReadBucket,
		storage.DiffWithExternalPaths(), // No need to set prefixes as the buckets are from the same location.
	)
	if err != nil {
		return err
	}
	diffExists := diffBuffer.Len() > 0
	defer func() {
		if retErr == nil && flags.ExitCode && diffExists {
			retErr = bufctl.ErrFileAnnotation
		}
	}()

	if flags.Diff {
		if diffExists {
			if _, err := io.Copy(container.Stdout(), diffBuffer); err != nil {
				return err
			}
		}
		// If we haven't overridden the output flag and haven't set write, we can stop here.
		if flags.Output == "-" && !flags.Write {
			return nil
		}
	}
	if flags.Write {
		changedPathSet := slicesext.ToStructMap(changedPaths)
		return storage.WalkReadObjects(
			ctx,
			formattedReadBucket,
			"",
			func(readObject storage.ReadObject) error {
				if _, ok := changedPathSet[readObject.Path()]; !ok {
					// no change, nothing to re-write
					return nil
				}
				// TODO FUTURE: This is a legacy hack that we shouldn't use. We should not
				// rely on external paths being writable.
				//
				// We do validation above on the flags.Write flag to quasi-ensure that ExternalPath
				// will be a real externalPath, but it's not great.
				file, err := os.OpenFile(readObject.ExternalPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					return err
				}
				defer func() {
					retErr = multierr.Append(retErr, file.Close())
				}()
				if _, err := file.ReadFrom(readObject); err != nil {
					return err
				}
				return nil
			},
		)
	}
	// Both flags.Diff and flags.Write not set, do output logic.
	switch t := dirOrProtoFileRef.(type) {
	case buffetch.DirRef:
		if err := writeToDir(ctx, flags.DisableSymlinks, formattedReadBucket, t); err != nil {
			return err
		}
	case buffetch.ProtoFileRef:
		if err := writeToProtoFile(ctx, container, formattedReadBucket, t); err != nil {
			return err
		}
	default:
		return syserror.Newf("buffetch ref type must be dir or proto file: %T", dirOrProtoFileRef)
	}
	return nil
}

func writeToDir(
	ctx context.Context,
	disableSymlinks bool,
	formattedReadBucket storage.ReadBucket,
	dirRef buffetch.DirRef,
) error {
	if err := createDirIfNotExists(dirRef.DirPath()); err != nil {
		return err
	}
	readWriteBucket, err := newStorageosProvider(disableSymlinks).NewReadWriteBucket(
		dirRef.DirPath(),
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	// We don't copy with ExternalPaths, we use Paths.
	// This is what we were always doing, including pre-refactor.
	_, err = storage.Copy(
		ctx,
		formattedReadBucket,
		readWriteBucket,
	)
	return err
}

func writeToProtoFile(
	ctx context.Context,
	container appext.Container,
	formattedReadBucket storage.ReadBucket,
	protoFileRef buffetch.ProtoFileRef,
) (retErr error) {
	writeCloser, err := buffetch.NewProtoFileWriter(container.Logger()).PutProtoFile(
		ctx,
		container,
		protoFileRef,
	)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeCloser.Close())
	}()
	return storage.WalkReadObjects(
		ctx,
		formattedReadBucket,
		"",
		func(readObject storage.ReadObject) error {
			data, err := io.ReadAll(readObject)
			if err != nil {
				return err
			}
			if _, err := writeCloser.Write(data); err != nil {
				return err
			}
			return nil
		},
	)
}

func createDirIfNotExists(dirPath string) error {
	// OK to use os.Stat instead of os.LStat here as this is CLI-only
	if _, err := os.Stat(dirPath); err != nil {
		// We don't need to check fileInfo.IsDir() because it's
		// already handled by the storageosProvider.
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return err
			}
			// We could os.RemoveAll if the overall command exits without error, but we're
			// not going to, just to be safe.
		}
	}
	return nil
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
		// We should have a better answer here. Right now, it's
		// possible that the other files in the same package are defined
		// in a remote dependency, which makes it impossible to rewrite
		// in-place.
		//
		// In the case that the user uses the -w flag, we'll either need
		// to return an error, or omit the file that it can't rewrite in-place
		// (potentially including a debug log).
		return appcmd.NewInvalidArgumentError("cannot specify include_package_files=true with format")
	}
	return nil
}

func newStorageosProvider(disableSymlinks bool) storageos.Provider {
	var options []storageos.ProviderOption
	if !disableSymlinks {
		options = append(options, storageos.ProviderWithSymlinks())
	}
	return storageos.NewProvider(options...)
}
