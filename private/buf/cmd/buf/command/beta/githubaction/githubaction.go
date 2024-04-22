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

package githubaction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/google/go-github/v61/github"
	"github.com/spf13/pflag"
)

const (
	formatFlagName          = "format"
	lintFlagName            = "lint"
	breakingFlagName        = "breaking"
	breakingAgainstFlagName = "breaking-against"
	pushFlagName            = "push"

	// Shared flags
	errorFormatFlagName     = "error-format"
	configFlagName          = "config"
	pathsFlagName           = "path"
	excludePathsFlagName    = "exclude-path"
	disableSymlinksFlagName = "disable-symlinks"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Run githubactions on Protobuf files.",
		Long:  bufcli.GetSourceOrModuleLong(`the source or module to get a price for`),
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
	ErrorFormat     string
	Config          string
	Paths           []string
	ExcludePaths    []string
	DisableSymlinks bool

	Format          bool
	Lint            bool
	Breaking        bool
	BreakingAgainst string
	Push            bool
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml file or data to use for configuration`,
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors or check violations printed to stdout. Must be one of %s",
			stringutil.SliceToString(buflint.AllFormatStrings),
		),
	)
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindPaths(flagSet, &f.Paths, pathsFlagName)
	bufcli.BindExcludePaths(flagSet, &f.ExcludePaths, excludePathsFlagName)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)

	flagSet.BoolVar(
		&f.Format,
		formatFlagName,
		false,
		"Format the Protobuf files",
	)
	flagSet.BoolVar(
		&f.Lint,
		lintFlagName,
		false,
		"Lint the Protobuf files",
	)
	flagSet.BoolVar(
		&f.Breaking,
		breakingFlagName,
		false,
		"Breaking change the Protobuf files",
	)
	flagSet.StringVar(
		&f.BreakingAgainst,
		breakingAgainstFlagName,
		"",
		"Breaking change the Protobuf files against the given commit",
	)
	flagSet.BoolVar(
		&f.Push,
		pushFlagName,
		false,
		"Push the Protobuf files",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	fmt.Println("running buf githubaction")
	for _, pair := range os.Environ() {
		fmt.Println(pair)
	}
	if err := bufcli.ValidateErrorFormatFlagLint(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	// Parse out if this is config-ignore-yaml.
	// This is messed.
	controllerErrorFormat := flags.ErrorFormat
	if controllerErrorFormat == "config-ignore-yaml" {
		controllerErrorFormat = "text"
	}
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
		bufctl.WithFileAnnotationErrorFormat(controllerErrorFormat),
		bufctl.WithFileAnnotationsToStdout(),
	)
	if err != nil {
		return err
	}
	workspace, err := controller.GetWorkspace(
		ctx,
		input,
		bufctl.WithTargetPaths(flags.Paths, flags.ExcludePaths),
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
	}

	imageWithConfigs, err := controller.GetTargetImageWithConfigs(
		ctx,
		input,
		bufctl.WithTargetPaths(flags.Paths, flags.ExcludePaths),
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
	}

	if flags.Format {
		if err := runFormat(ctx, container, workspace); err != nil {
			return err
		}
	}
	if flags.Lint {
		if err := runLint(ctx, flags, container, imageWithConfigs); err != nil {
			return err
		}
	}
	client := github.NewClient(nil)
	// list all organizations for user "willnorris"
	orgs, _, err := client.Organizations.List(ctx, "willnorris", nil)
	if err != nil {
		return err
	}
	for _, org := range orgs {
		fmt.Println(org.GetLogin())
	}
	return fmt.Errorf("not implemented")
}

func runFormat(
	ctx context.Context,
	container appext.Container,
	workspace bufworkspace.Workspace,
) error {
	fmt.Println("formatting image")
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
	runner := command.NewRunner() // TODO: unify?
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
	if diffBuffer.Len() > 0 {
		if _, err := io.Copy(container.Stdout(), diffBuffer); err != nil {
			return err
		}
		return bufctl.ErrFileAnnotation
	}
	return nil
}

func runLint(
	ctx context.Context,
	flags *flags,
	container appext.Container,
	imageWithConfigs []bufctl.ImageWithConfig,
) error {
	fmt.Println("linting image")
	var allFileAnnotations []bufanalysis.FileAnnotation
	for _, imageWithConfig := range imageWithConfigs {
		if err := buflint.NewHandler(
			container.Logger(),
			tracing.NewTracer(container.Tracer()),
		).Check(
			ctx,
			imageWithConfig.LintConfig(),
			imageWithConfig,
		); err != nil {
			var fileAnnotationSet bufanalysis.FileAnnotationSet
			if errors.As(err, &fileAnnotationSet) {
				allFileAnnotations = append(allFileAnnotations, fileAnnotationSet.FileAnnotations()...)
			} else {
				return err
			}
		}
	}
	hasErrors := len(allFileAnnotations) > 0
	if hasErrors {
		allFileAnnotationSet := bufanalysis.NewFileAnnotationSet(allFileAnnotations...)
		if flags.ErrorFormat == "config-ignore-yaml" {
			if err := buflint.PrintFileAnnotationSetConfigIgnoreYAMLV1(
				container.Stdout(),
				allFileAnnotationSet,
			); err != nil {
				return err
			}
		} else {
			if err := bufanalysis.PrintFileAnnotationSet(
				container.Stdout(),
				allFileAnnotationSet,
				flags.ErrorFormat,
			); err != nil {
				return err
			}
		}
		return bufctl.ErrFileAnnotation
	}
	return nil
}
