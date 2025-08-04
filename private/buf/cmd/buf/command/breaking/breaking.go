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

package breaking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"buf.build/go/standard/xstrings"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/wasm"
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
	againstRegistryFlagName   = "against-registry"
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
		Long: `This command makes sure that the <input> location has no breaking changes compared to the <against-input> location.

` +
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
	AgainstRegistry   bool
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
			xstrings.SliceToString(bufanalysis.AllFormatStrings),
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
			`Required, except if --%s is set. The source, module, or image to check against. Must be one of format %s`,
			againstRegistryFlagName,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&f.AgainstConfig,
		againstConfigFlagName,
		"",
		`The buf.yaml file or data to use to configure the against source, module, or image`,
	)
	flagSet.BoolVar(
		&f.AgainstRegistry,
		againstRegistryFlagName,
		false,
		fmt.Sprintf(
			`Run breaking checks against the latest commit on the default branch in the registry. All modules in the input must have a name configured, otherwise this will fail.
If a remote module is not found with the configured name, then this will fail. This cannot be set with --%s.`,
			againstFlagName,
		),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	if err := validateFlags(flags); err != nil {
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
	wasmRuntime, err := bufcli.NewWasmRuntime(ctx, container)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, wasmRuntime.Close(ctx))
	}()
	// Do not exclude imports here. bufcheck's Client requires all imports.
	// Use bufcheck's BreakingWithExcludeImports.
	imageWithConfigs, checkClient, err := controller.GetTargetImageWithConfigsAndCheckClient(
		ctx,
		input,
		wasmRuntime,
		bufctl.WithTargetPaths(flags.Paths, flags.ExcludePaths),
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
	var againstImages []bufimage.Image
	if flags.Against != "" {
		// Do not exclude imports here. bufcheck's Client requires all imports.
		// Use bufcheck's BreakingWithExcludeImports.
		againstImagesWithConfigs, _, err := controller.GetTargetImageWithConfigsAndCheckClient(
			ctx,
			flags.Against,
			wasm.UnimplementedRuntime,
			bufctl.WithTargetPaths(externalPaths, flags.ExcludePaths),
			bufctl.WithConfigOverride(flags.AgainstConfig),
		)
		if err != nil {
			return err
		}
		// We do not require the check configs from the against target once built, so they can
		// be dropped here.
		againstImages, err = xslices.MapError(
			againstImagesWithConfigs,
			func(imageWithConfig bufctl.ImageWithConfig) (bufimage.Image, error) {
				againstImage, ok := imageWithConfig.(bufimage.Image)
				if !ok {
					return nil, syserror.New("imageWithConfig could not be converted to Image")
				}
				return againstImage, nil
			},
		)
		if err != nil {
			return err
		}
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	moduleServiceProvider := bufregistryapimodule.NewClientProvider(clientConfig)
	if flags.AgainstRegistry {
		for _, imageWithConfig := range imageWithConfigs {
			moduleFullName := imageWithConfig.ModuleFullName()
			if moduleFullName == nil {
				if imageWithConfig.ModuleOpaqueID() == "" {
					// This can occur in the case of a [buffetch.MessageRef], where we resolve the message
					// ref directly from the bucket without building the [bufmodule.Module]. In that case,
					// we are unnable to use --against-registry.
					return fmt.Errorf("cannot use --%s with unnamed module", againstRegistryFlagName)
				}
				return fmt.Errorf(
					"cannot use --%s with unnamed module, %s",
					againstRegistryFlagName,
					imageWithConfig.ModuleOpaqueID(),
				)
			}
			// Check commits exist for the module on the default label before we try to get the against image.
			// TODO: remove this check once we have error details to indicate that the against input
			// has no commits on the default label. We use the API directly here to avoid issues of conflating
			// dependency errors to empty module errors.
			commitServiceClient := moduleServiceProvider.V1CommitServiceClient(moduleFullName.Registry())
			if _, err := commitServiceClient.GetCommits(
				ctx,
				connect.NewRequest(
					&modulev1.GetCommitsRequest{
						ResourceRefs: []*modulev1.ResourceRef{
							{
								Value: &modulev1.ResourceRef_Name_{
									Name: &modulev1.ResourceRef_Name{
										Owner:  moduleFullName.Owner(),
										Module: moduleFullName.Name(),
									},
								},
							},
						},
					},
				),
			); err != nil {
				if connect.CodeOf(err) == connect.CodeFailedPrecondition {
					// This error occurs when the against input is a module that has no commits on the default branch.
					// We ignore this case to support new modulues that have just been created in the registry.
					container.Logger().DebugContext(
						ctx, "ignoring empty module without commits on the default branch",
						slog.String("module", imageWithConfig.ModuleFullName().String()),
					)
					continue
				}
				return err
			}
			againstImage, err := controller.GetImage(
				ctx,
				moduleFullName.String(),
				bufctl.WithTargetPaths(externalPaths, flags.ExcludePaths),
				bufctl.WithConfigOverride(flags.AgainstConfig),
			)
			if err != nil {
				return err
			}
			againstImages = append(againstImages, againstImage)
		}
	}
	if len(imageWithConfigs) != len(againstImages) {
		// In the case where the input and against workspaces do not contain the same number of
		// images, this could happen if the input contains new module(s). However, we require
		// the number of images to match because of module-specific [bufconfig.BreakingConfig].
		// This can result in a less satisfying UX when adding modules to a workspace.
		//
		// To mitigate this for users adding new modules to their workspace, for the case where
		// len(imageWithConfigs) > len(againstImages), if all modules in [imageWithConfigs] have
		// the same [bufconfig.BreakingConfig] (so no unique, module-specific [bufconfig.BreakingConfig]),
		// we query the [againstImages] for the matching modules and ignore any modules from
		// [imageWithConfigs] that are not found in [againstImages].
		//
		// In the case where len(imageWithConfigs) < len(againstImages) or there are module-specific
		// [bufconfig.BreakingConfig], we still return an error. Also, if the roots change, we're
		// torched. (Issue #3641)
		if len(imageWithConfigs) > len(againstImages) && hasNoUniqueBreakingConfig(imageWithConfigs) {
			imageWithConfigs, err = filterImageWithConfigsNotInAgainstImages(imageWithConfigs, againstImages)
			if err != nil {
				return err
			}
		} else {
			return newInputAgainstImageCountError(len(imageWithConfigs), len(againstImages))
		}
	}
	// We add all check configs (both lint and breaking) as related configs to check if plugins
	// have rules configured.
	// We allocated twice the size of imageWithConfigs for both lint and breaking configs.
	allCheckConfigs := make([]bufconfig.CheckConfig, 0, len(imageWithConfigs)*2)
	for _, imageWithConfig := range imageWithConfigs {
		allCheckConfigs = append(allCheckConfigs, imageWithConfig.LintConfig())
		allCheckConfigs = append(allCheckConfigs, imageWithConfig.BreakingConfig())
	}
	var allFileAnnotations []bufanalysis.FileAnnotation
	for i, imageWithConfig := range imageWithConfigs {
		breakingOptions := []bufcheck.BreakingOption{
			bufcheck.WithPluginConfigs(imageWithConfig.PluginConfigs()...),
			bufcheck.WithPolicyConfigs(imageWithConfig.PolicyConfigs()...),
			bufcheck.WithRelatedCheckConfigs(allCheckConfigs...),
		}
		if flags.ExcludeImports {
			breakingOptions = append(breakingOptions, bufcheck.BreakingWithExcludeImports())
		}
		if err := checkClient.Breaking(
			ctx,
			imageWithConfig.BreakingConfig(),
			imageWithConfig,
			againstImages[i],
			breakingOptions...,
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
	return xslices.MapKeysToSlice(externalPaths), nil
}

func validateFlags(flags *flags) error {
	if flags.Against == "" && !flags.AgainstRegistry {
		return fmt.Errorf("Must set --%s or --%s", againstFlagName, againstRegistryFlagName)
	}
	if flags.Against != "" && flags.AgainstRegistry {
		return fmt.Errorf("Cannot set both --%s and --%s", againstFlagName, againstRegistryFlagName)
	}
	return nil
}

// hasNoUniqueBreakingConfig iterates through imageWithConfigs and checks to see if there
// are any unique [bufconfig.BreakingConfig]. It returns true if all [bufconfig.BreakingConfig]
// are the same across all the images.
func hasNoUniqueBreakingConfig(imageWithConfigs []bufctl.ImageWithConfig) bool {
	var base bufconfig.BreakingConfig
	for _, imageWithConfig := range imageWithConfigs {
		if base == nil {
			base = imageWithConfig.BreakingConfig()
			continue
		}
		if !equalBreakingConfig(base, imageWithConfig.BreakingConfig()) {
			return false
		}
		base = imageWithConfig.BreakingConfig()
	}
	return true
}

// Checks if the specified [bufconfig.BreakingConfig]s are equal. Returns true if both
// [bufconfig.BreakingConfig] have the same configuration parameters.
func equalBreakingConfig(breakingConfig1, breakingConfig2 bufconfig.BreakingConfig) bool {
	if breakingConfig1.Disabled() == breakingConfig2.Disabled() &&
		breakingConfig1.FileVersion() == breakingConfig2.FileVersion() &&
		slices.Equal(breakingConfig1.UseIDsAndCategories(), breakingConfig2.UseIDsAndCategories()) &&
		slices.Equal(breakingConfig1.ExceptIDsAndCategories(), breakingConfig2.ExceptIDsAndCategories()) &&
		slices.Equal(breakingConfig1.IgnorePaths(), breakingConfig2.IgnorePaths()) &&
		maps.EqualFunc(
			breakingConfig1.IgnoreIDOrCategoryToPaths(),
			breakingConfig2.IgnoreIDOrCategoryToPaths(),
			slices.Equal[[]string],
		) &&
		breakingConfig1.DisableBuiltin() == breakingConfig2.DisableBuiltin() &&
		breakingConfig1.IgnoreUnstablePackages() == breakingConfig2.IgnoreUnstablePackages() {
		return true
	}
	return false
}

// A helper function for filtering out [bufctl.ImageWithConfig]s from [imagesWithConfig]
// if there is no corresponding image in [againstImages]. We determine this based on image
// file path.
//
// This assumes that len(imageWithConfigs) > len(againstImages).
// We also expect that each image in [againstImages] is mapped only once to a single
// imageWithConfig in [imagesWithConfig]. If an againstImage is found, then we don't check
// it again. We also validate that each image in [againstImages] is mapped to an imageWithConfig
// from [imageWithConfigs].
func filterImageWithConfigsNotInAgainstImages(
	imageWithConfigs []bufctl.ImageWithConfig,
	againstImages []bufimage.Image,
) ([]bufctl.ImageWithConfig, error) {
	foundAgainstImageIndices := make(map[int]struct{})
	var filteredImageWithConfigs []bufctl.ImageWithConfig
	for _, imageWithConfig := range imageWithConfigs {
		for _, imageFile := range imageWithConfig.Files() {
			var foundImage bufimage.Image
			for i, againstImage := range againstImages {
				if _, ok := foundAgainstImageIndices[i]; ok {
					continue
				}
				if againstImage.GetFile(imageFile.Path()) != nil {
					foundAgainstImageIndices[i] = struct{}{}
					foundImage = againstImage
					break
				}
			}
			if foundImage != nil {
				filteredImageWithConfigs = append(filteredImageWithConfigs, imageWithConfig)
				break
			}
		}
	}
	// If we are unsuccessful in mapping all againstImages to a unique imageWithConfig, then
	// we return the same error message.
	if len(foundAgainstImageIndices) != len(againstImages) || len(againstImages) != len(filteredImageWithConfigs) {
		return nil, newInputAgainstImageCountError(len(imageWithConfigs), len(againstImages))
	}
	return filteredImageWithConfigs, nil
}

func newInputAgainstImageCountError(lenImageWithConfigs, lenAgainstImages int) error {
	return fmt.Errorf(
		"input contained %d images, whereas against contained %d images",
		lenImageWithConfigs,
		lenAgainstImages,
	)
}
