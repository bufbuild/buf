// Copyright 2020-2023 Buf Technologies, Inc.
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	"github.com/bufbuild/buf/private/pkg/stringutil"
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
	container appext.Container,
	flags *flags,
) (retErr error) {
	if flags.Output != "-" && flags.Write {
		return fmt.Errorf("--%s cannot be used with --%s", outputFlagName, writeFlagName)
	}
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	outputSourceOrModuleRef, err := buffetch.NewRefParser(logger).GetSourceRef(ctx, flags.Output)
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
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace),
	)

	fileInfos, err := bufmodule.GetFileInfos(ctx, moduleReadBucket)
	if err != nil {
		return err
	}
	// An output with a .proto extension could be an output directory but we're going to ignore that
	// as that's borderline pathological for formatting.
	if filepath.Ext(flags.Output) == ".proto" && len(fileInfos) > 1 {
		externalPaths := slicesext.Map(fileInfos, func(fileInfo bufmodule.FileInfo) string { return fileInfo.ExternalPath() })
		return appcmd.NewInvalidArgumentErrorf("--%s specified as single .proto file but multiple files targeted for formatting: %s", flags.Output, strings.Join(externalPaths, ","))
	}

	originalReadBucket := bufmodule.ModuleReadBucketToStorageReadBucket(moduleReadBucket)
	//paths, err := storage.AllPaths(ctx, originalReadBucket, "")
	//if err != nil {
	//return err
	//}
	//fmt.Println("original:\n" + strings.Join(paths, "\n") + "\n")
	formattedReadBucket, err := bufformat.FormatBucket(ctx, originalReadBucket)
	if err != nil {
		return err
	}

	diffBuffer := bytes.NewBuffer(nil)
	if err := storage.Diff(
		ctx,
		runner,
		diffBuffer,
		originalReadBucket,
		formattedReadBucket,
		storage.DiffWithExternalPaths(), // No need to set prefixes as the buckets are from the same location.
	); err != nil {
		return err
	}
	diffExists := diffBuffer.Len() > 0

	if flags.Diff && diffExists {
		if _, err := io.Copy(container.Stdout(), diffBuffer); err != nil {
			return err
		}
	}
	if flags.Rewrite {
		if err := storage.WalkReadObjects(
			ctx,
			formattedReadBucket,
			"",
			func(readObject storage.ReadObject) error {
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
	if flags.ExitCode && diffExists {
		return bufctl.ErrFileAnnotation
	}
	return nil
}
