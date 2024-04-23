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

package breaking

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName       = "error-format"
	excludeImportsFlagName    = "exclude-imports"
	pathsFlagName             = "path"
	limitToInputFilesFlagName = "limit-to-input-files"
	configFlagName            = "config"
	againstFlagName           = "against"
	againstConfigFlagName     = "against-config"
	excludePathsFlagName      = "exclude-path"
	disableSymlinksFlagName   = "disable-symlinks"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input> --against <against-input>",
		Short: "Verify no breaking changes have been made",
		Long: `buf breaking makes sure that the <input> location has no breaking changes compared to the <against-input> location. ` +
			bufcli.GetInputLong(`the source, module, or image to check for breaking changes`),
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
	ErrorFormat       string
	ExcludeImports    bool
	LimitToInputFiles bool
	Paths             []string
	Config            string
	Against           string
	AgainstConfig     string
	ExcludePaths      []string
	DisableSymlinks   bool
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindPaths(flagSet, &f.Paths, pathsFlagName)
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindExcludePaths(flagSet, &f.ExcludePaths, excludePathsFlagName)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors or check violations printed to stdout. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.BoolVar(
		&f.ExcludeImports,
		excludeImportsFlagName,
		false,
		"Exclude imports from breaking change detection.",
	)
	flagSet.BoolVar(
		&f.LimitToInputFiles,
		limitToInputFilesFlagName,
		false,
		fmt.Sprintf(
			`Only run breaking checks against the files in the input
When set, the against input contains only the files in the input
Overrides --%s`,
			pathsFlagName,
		),
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml file or data to use for configuration`,
	)
	flagSet.StringVar(
		&f.Against,
		againstFlagName,
		"",
		fmt.Sprintf(
			`Required. The source, module, or image to check against. Must be one of format %s`,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&f.AgainstConfig,
		againstConfigFlagName,
		"",
		`The buf.yaml file or data to use to configure the against source, module, or image`,
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	if err := bufcli.ValidateRequiredFlag(againstFlagName, flags.Against); err != nil {
		return err
	}
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
		bufctl.WithFileAnnotationErrorFormat(flags.ErrorFormat),
		bufctl.WithFileAnnotationsToStdout(),
	)
	if err != nil {
		return err
	}
	imageWithConfigs, err := controller.GetTargetImageWithConfigs(
		ctx,
		input,
		bufctl.WithTargetPaths(flags.Paths, flags.ExcludePaths),
		bufctl.WithImageExcludeImports(flags.ExcludeImports),
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
	}
	// TODO: this doesn't actually work because we're using the same file paths for both sides
	// of the roots change, then we're torched
	externalPaths := flags.Paths
	if flags.LimitToInputFiles {
		externalPaths, err = getExternalPathsForImages(imageWithConfigs)
		if err != nil {
			return err
		}
	}
	againstImageWithConfigs, err := controller.GetTargetImageWithConfigs(
		ctx,
		flags.Against,
		bufctl.WithTargetPaths(externalPaths, flags.ExcludePaths),
		bufctl.WithImageExcludeImports(flags.ExcludeImports),
		bufctl.WithConfigOverride(flags.AgainstConfig),
	)
	if err != nil {
		return err
	}
	if len(imageWithConfigs) != len(againstImageWithConfigs) {
		// If workspaces are being used as input, the number
		// of images MUST match. Otherwise the results will
		// be meaningless and yield false positives.
		//
		// And similar to the note above, if the roots change,
		// we're torched.
		return fmt.Errorf(
			"input contained %d images, whereas against contained %d images",
			len(imageWithConfigs),
			len(againstImageWithConfigs),
		)
	}
	var allFileAnnotations []bufanalysis.FileAnnotation
	for i, imageWithConfig := range imageWithConfigs {
		if err := bufbreaking.NewHandler(
			container.Logger(),
			tracing.NewTracer(container.Tracer()),
		).Check(
			ctx,
			imageWithConfig.BreakingConfig(),
			againstImageWithConfigs[i],
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
	if len(allFileAnnotations) > 0 {
		allFileAnnotationSet := bufanalysis.NewFileAnnotationSet(allFileAnnotations...)
		if err := bufanalysis.PrintFileAnnotationSet(
			container.Stdout(),
			allFileAnnotationSet,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return bufctl.ErrFileAnnotation
	}
	return nil
}

func getExternalPathsForImages[I bufimage.Image, S ~[]I](images S) ([]string, error) {
	externalPaths := make(map[string]struct{})
	for _, image := range images {
		for _, imageFile := range image.Files() {
			externalPaths[imageFile.ExternalPath()] = struct{}{}
		}
	}
	return slicesext.MapKeysToSlice(externalPaths), nil
}
