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
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/buf/bufwork"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
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
	builder appflag.Builder,
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
		Args: cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
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
			buffetch.SourceFormatsString,
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
	container appflag.Container,
	flags *flags,
) (retErr error) {
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	if flags.Output != "-" && flags.Write {
		return fmt.Errorf("--%s cannot be used with --%s", outputFlagName, writeFlagName)
	}
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	refParser := buffetch.NewRefParser(container.Logger())
	sourceOrModuleRef, err := refParser.GetSourceOrModuleRef(ctx, source)
	if err != nil {
		return err
	}
	if _, ok := sourceOrModuleRef.(buffetch.ModuleRef); ok && flags.Write {
		return fmt.Errorf("--%s cannot be used with module reference inputs", writeFlagName)
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	moduleReader, err := bufcli.NewModuleReaderAndCreateCacheDirs(container, clientConfig)
	if err != nil {
		return err
	}
	runner := command.NewRunner()
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	moduleConfigReader, err := bufcli.NewWireModuleConfigReaderForModuleReader(
		container,
		storageosProvider,
		runner,
		clientConfig,
		moduleReader,
	)
	if err != nil {
		return err
	}
	moduleConfigSet, err := moduleConfigReader.GetModuleConfigSet(
		ctx,
		container,
		sourceOrModuleRef,
		flags.Config,
		flags.Paths,
		flags.ExcludePaths,
		false,
	)
	if err != nil {
		return err
	}
	moduleConfigs := moduleConfigSet.ModuleConfigs()
	var outputDirectory string
	var singleFileOutputFilename string
	if flags.Output != "-" {
		// The output file type is determined based on its extension,
		// so it's possible to write a single file's formatted content
		// to another single file.
		//
		//  $ buf format simple.proto -o simple.formatted.proto
		//
		// In this case, it's also possible to write an entire directory's
		// formatted content to a single file (like we see in the default
		// behavior with stdout).
		//
		//  $ buf format simple -o simple.formatted.proto
		//
		outputRef, err := refParser.GetSourceOrModuleRef(ctx, flags.Output)
		if err != nil {
			return err
		}
		if _, ok := outputRef.(buffetch.ProtoFileRef); ok {
			if directory := filepath.Dir(flags.Output); directory != "." {
				// The output is a single file, so we need to create
				// the file's directory (if any).
				//
				// For example,
				//
				//  $ buf format simple.proto -o formatted/simple.formatted.proto
				//
				outputDirectory = directory
			}
			// The outputDirectory will not be set for single file outputs
			// in the current directory (e.g. simple.formatted.proto).
			singleFileOutputFilename = flags.Output
		} else {
			// The output is a directory, so we can just create it as-is.
			outputDirectory = flags.Output
		}
	}
	if protoFileRef, ok := sourceOrModuleRef.(buffetch.ProtoFileRef); ok {
		// If we have a single ProtoFileRef, we only want to format that file.
		// The file will be available from the first module (i.e. it's
		// the target source, or the first module in a workspace).
		if len(moduleConfigs) == 0 {
			// Unreachable - we should always have at least one module.
			return fmt.Errorf("could not build module for %s", container.Arg(0))
		}
		if protoFileRef.IncludePackageFiles() {
			// TODO: We need to have a better answer here. Right now, it's
			// possible that the other files in the same package are defined
			// in a remote dependency, which makes it impossible to rewrite
			// in-place.
			//
			// In the case that the user uses the -w flag, we'll either need
			// to return an error, or omit the file that it can't rewrite in-place
			// (potentially including a debug log).
			return errors.New("this command does not support including package files")
		}
		module := moduleConfigs[0].Module()
		fileInfos, err := module.TargetFileInfos(ctx)
		if err != nil {
			return err
		}
		var moduleFile bufmodule.ModuleFile
		for _, fileInfo := range fileInfos {
			if _, err := protoFileRef.PathForExternalPath(fileInfo.ExternalPath()); err != nil {
				// The target file we're looking for is the only one that will not
				// return an error.
				continue
			}
			moduleFile, err = module.GetModuleFile(
				ctx,
				fileInfo.Path(),
			)
			if err != nil {
				return err
			}
			defer func() {
				retErr = multierr.Append(retErr, moduleFile.Close())
			}()
			break
		}
		if moduleFile == nil {
			// This will only happen if a buf.work.yaml exists in a parent
			// directory, but it does not contain the target file.
			//
			// This is also a problem for other commands that interact
			// with buffetch.ProtoFileRef.
			//
			// TODO: Fix the buffetch.ProtoFileRef so that it works in
			// these situtations.
			return fmt.Errorf(
				"source %s was not found - is the directory containing this file defined in your %s?",
				container.Arg(0),
				bufwork.ExternalConfigV1FilePath,
			)
		}
		module, err = bufmodule.ModuleWithTargetPaths(
			module,
			[]string{
				moduleFile.Path(),
			},
			nil, // Nothing to exclude.
		)
		if err != nil {
			return err
		}
		diffPresent, err := formatModule(
			ctx,
			container,
			runner,
			storageosProvider,
			module,
			outputDirectory,
			singleFileOutputFilename,
			flags.ErrorFormat,
			flags.Diff,
			flags.Write,
		)
		if err != nil {
			return err
		}
		if flags.ExitCode && diffPresent {
			return bufcli.ErrFileAnnotation
		}
		return nil
	}
	for _, moduleConfig := range moduleConfigs {
		diffPresent, err := formatModule(
			ctx,
			container,
			runner,
			storageosProvider,
			moduleConfig.Module(),
			outputDirectory,
			singleFileOutputFilename,
			flags.ErrorFormat,
			flags.Diff,
			flags.Write,
		)
		if err != nil {
			return err
		}
		if flags.ExitCode && diffPresent {
			return bufcli.ErrFileAnnotation
		}
	}
	return nil
}

// formatModule formats the module's target files and writes them to the
// writeBucket, if any. If diff is true, the diff between the original and
// formatted files is written to stdout.
//
// Returns true if there was a diff and no other error.
func formatModule(
	ctx context.Context,
	container appflag.Container,
	runner command.Runner,
	storageosProvider storageos.Provider,
	module bufmodule.Module,
	outputDirectory string,
	singleFileOutputFilename string,
	errorFormat string,
	diff bool,
	rewrite bool,
) (_ bool, retErr error) {
	originalReadWriteBucket := storagemem.NewReadWriteBucket()
	if err := bufmodule.TargetModuleFilesToBucket(
		ctx,
		module,
		originalReadWriteBucket,
	); err != nil {
		return false, err
	}
	// Note that external paths are set properly for the files in this read bucket.
	formattedReadBucket, err := bufformat.FormatModule(ctx, module)
	if err != nil {
		return false, err
	}
	diffBuffer := bytes.NewBuffer(nil)
	if err := storage.Diff(
		ctx,
		runner,
		diffBuffer,
		originalReadWriteBucket,
		formattedReadBucket,
		storage.DiffWithExternalPaths(), // No need to set prefixes as the buckets are from the same location.
	); err != nil {
		return false, err
	}
	diffPresent := diffBuffer.Len() > 0
	if diff {
		if _, err := io.Copy(container.Stdout(), diffBuffer); err != nil {
			return false, err
		}
		if outputDirectory == "" && singleFileOutputFilename == "" && !rewrite {
			// If the user specified --diff and has not explicitly overridden
			// the --output or rewritten the sources in-place with --write, we
			// can stop here.
			return diffPresent, nil
		}
	}
	if rewrite {
		// Rewrite the sources in place.
		if err := storage.WalkReadObjects(
			ctx,
			originalReadWriteBucket,
			"",
			func(readObject storage.ReadObject) error {
				formattedReadObject, err := formattedReadBucket.Get(ctx, readObject.Path())
				if err != nil {
					return err
				}
				// We use os.OpenFile here instead of storage.Copy for a few reasons.
				//
				// storage.Copy operates on normal paths, so the copied content is always placed
				// relative to the bucket's root (as expected). The rewrite in-place behavior can
				// be rephrased as writing to the same bucket as the input (e.g. buf format proto -o proto).
				//
				// Now, if the user asks to rewrite an entire workspace (i.e. a directory containing
				// a buf.work.yaml), we would need to call storage.Copy for each of the directories
				// defined in the workspace. This involves parsing the buf.work.yaml and creating
				// a storage.Bucket for each of the directories.
				//
				// It's simpler to just copy the files in-place based on their external path since
				// it's the same behavior for single files, directories, and workspaces.
				file, err := os.OpenFile(readObject.ExternalPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					return err
				}
				defer func() {
					retErr = multierr.Append(retErr, file.Close())
				}()
				if _, err := file.ReadFrom(formattedReadObject); err != nil {
					return err
				}
				return nil
			},
		); err != nil {
			return false, err
		}
		return diffPresent, nil
	}
	var readWriteBucket storage.ReadWriteBucket
	if outputDirectory != "" {
		// OK to use os.Stat instead of os.LStat here as this is CLI-only
		if _, err := os.Stat(outputDirectory); err != nil {
			// We don't need to check fileInfo.IsDir() because it's
			// already handled by the storageosProvider.
			if os.IsNotExist(err) {
				if err := os.MkdirAll(outputDirectory, 0755); err != nil {
					return false, err
				}
				// Although unlikely, if an error occurs in the midst of
				// writing the formatted files, we want to clean up the
				// directory we just created because it didn't previously
				// exist.
				defer func() {
					if retErr != nil {
						retErr = multierr.Append(retErr, os.RemoveAll(outputDirectory))
					}
				}()
			}
		}
		readWriteBucket, err = storageosProvider.NewReadWriteBucket(
			outputDirectory,
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return false, err
		}
	}
	if readWriteBucket == nil || singleFileOutputFilename != "" {
		// If the readWriteBucket is nil, we write the output to stdout.
		//
		// If a single file output was used, we can't just copy the content
		// between buckets - we need to write all of the bucket's content
		// into the single file (exactly like we do for writing to stdout).
		//
		// We might want to order these, although the output is kind of useless
		// if we're writing more than one file.
		writer := container.Stdout()
		if singleFileOutputFilename != "" {
			file, err := os.OpenFile(singleFileOutputFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return false, err
			}
			defer func() {
				retErr = multierr.Append(retErr, file.Close())
			}()
			writer = file
		}
		if err := storage.WalkReadObjects(
			ctx,
			formattedReadBucket,
			"",
			func(readObject storage.ReadObject) error {
				data, err := io.ReadAll(readObject)
				if err != nil {
					return err
				}
				if _, err := writer.Write(data); err != nil {
					return err
				}
				return nil
			},
		); err != nil {
			return false, err
		}
		return diffPresent, nil
	}
	// The user specified -o, so we copy the files into the output bucket.
	if _, err := storage.Copy(
		ctx,
		formattedReadBucket,
		readWriteBucket,
	); err != nil {
		return false, err
	}
	return diffPresent, nil
}
