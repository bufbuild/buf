// Copyright 2020-2022 Buf Technologies, Inc.
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
	"context"
	"fmt"
	"io"
	"os"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufformat"
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
		Use:   name + " <input>",
		Short: "Format all Protobuf files from the specified input and output the result.",
		Long:  bufcli.GetInputLong(`the source or module to format`),
		Args:  cobra.MaximumNArgs(1),
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
		"Display diffs instead of rewriting files.",
	)
	flagSet.BoolVarP(
		&f.Write,
		writeFlagName,
		writeFlagShortName,
		false,
		"Rewrite files in-place.",
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"-",
		fmt.Sprintf(
			`The output location for the formatted files. Must be one of format %s. If omitted, the result is written to stdout.`,
			buffetch.SourceFormatsString,
		),
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The file or data to use for configuration.`,
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
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	sourceOrModuleRef, err := buffetch.NewRefParser(container.Logger(), buffetch.RefParserWithProtoFileRefAllowed()).GetSourceOrModuleRef(ctx, input)
	if err != nil {
		return err
	}
	if _, ok := sourceOrModuleRef.(buffetch.ModuleRef); ok && flags.Write {
		return fmt.Errorf("--%s cannot be used with module reference inputs", writeFlagName)
	}
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	moduleReader, err := bufcli.NewModuleReaderAndCreateCacheDirs(container, registryProvider)
	if err != nil {
		return err
	}
	runner := command.NewRunner()
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	moduleConfigReader, err := bufcli.NewWireModuleConfigReaderForModuleReader(
		container,
		storageosProvider,
		runner,
		registryProvider,
		moduleReader,
	)
	if err != nil {
		return err
	}
	moduleConfigs, err := moduleConfigReader.GetModuleConfigs(
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
	var readWriteBucket storage.ReadWriteBucket
	if flags.Output != "-" {
		if err := os.MkdirAll(flags.Output, 0755); err != nil {
			return err
		}
		readWriteBucket, err = storageosProvider.NewReadWriteBucket(
			flags.Output,
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return err
		}
	}
	if protoFileRef, ok := sourceOrModuleRef.(buffetch.ProtoFileRef); ok {
		// If we have a single ProtoFileRef, we only want to format that file.
		// The file will be available from the first module (i.e. it's
		// the target input, or the first module in a workspace).
		if len(moduleConfigs) == 0 {
			// Unreachable - we should always have at least one module.
			return fmt.Errorf("could not build module for %s", container.Arg(0))
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
			return fmt.Errorf("input %s was not found - is the directory containing this file defined in your buf.work.yaml?", container.Arg(0))
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
		return formatModule(
			ctx,
			container,
			runner,
			module,
			readWriteBucket,
			flags.ErrorFormat,
			flags.Diff,
			flags.Write,
		)
	}
	for _, moduleConfig := range moduleConfigs {
		if err := formatModule(
			ctx,
			container,
			runner,
			moduleConfig.Module(),
			readWriteBucket,
			flags.ErrorFormat,
			flags.Diff,
			flags.Write,
		); err != nil {
			return err
		}
	}
	return nil
}

// formatModule formats the module's target files and writes them to the
// writeBucket, if any. If diff is true, the diff between the original and
// formatted files is written to stdout.
func formatModule(
	ctx context.Context,
	container appflag.Container,
	runner command.Runner,
	module bufmodule.Module,
	writeBucket storage.WriteBucket,
	errorFormat string,
	diff bool,
	rewrite bool,
) (retErr error) {
	// Note that external paths are set properly for the files in this read bucket.
	formattedReadBucket, err := bufformat.Format(ctx, module)
	if err != nil {
		return err
	}
	var originalReadWriteBucket storage.ReadWriteBucket
	if diff || rewrite {
		originalReadWriteBucket = storagemem.NewReadWriteBucket()
		if err := bufmodule.TargetModuleFilesToBucket(
			ctx,
			module,
			originalReadWriteBucket,
		); err != nil {
			return err
		}
	}
	if diff {
		if err := storage.Diff(
			ctx,
			runner,
			container.Stdout(),
			originalReadWriteBucket,
			formattedReadBucket,
			storage.DiffWithExternalPaths(), // No need to set prefixes as the buckets are from the same location.
		); err != nil {
			return err
		}
		if writeBucket == nil || !rewrite {
			// If the user specified --diff and has not explicitly overridden
			// the --output or rewritten the sources in-place with --write, we
			// can stop here.
			return nil
		}
	}
	if rewrite {
		// Rewrite the sources in place.
		return storage.WalkReadObjects(
			ctx,
			originalReadWriteBucket,
			"",
			func(readObject storage.ReadObject) error {
				formattedReadObject, err := formattedReadBucket.Get(ctx, readObject.Path())
				if err != nil {
					return err
				}
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
		)
	}
	if writeBucket == nil {
		// If the writeBucket is nil, we write the output to stdout.
		//
		// We might want to order these, although the output is kind of useless
		// if we're writing more than one file to stdout.
		return storage.WalkReadObjects(
			ctx,
			formattedReadBucket,
			"",
			func(readObject storage.ReadObject) error {
				data, err := io.ReadAll(readObject)
				if err != nil {
					return err
				}
				if _, err := container.Stdout().Write(data); err != nil {
					return err
				}
				return nil
			},
		)
	}
	// The user specified -o, so we copy the files into the output bucket.
	if _, err := storage.Copy(
		ctx,
		formattedReadBucket,
		writeBucket,
	); err != nil {
		return err
	}
	return nil
}
