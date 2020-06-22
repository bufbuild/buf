// Copyright 2020 Buf Technologies, Inc.
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

package buf

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
)

func imageBuild(ctx context.Context, container *container) (retErr error) {
	if container.Output == "" {
		return fmt.Errorf("--%s is required", imageBuildOutputFlagName)
	}
	asJSON, err := internal.IsFormatJSON(errorFormatFlagName, container.ErrorFormat)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufcliEnvReader(
		container.Logger(),
		imageBuildInputFlagName,
		imageBuildConfigFlagName,
		// must be source only
	).GetSourceEnv(
		ctx,
		container,
		container.Input,
		container.Config,
		container.Files,
		false,
		container.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdout potentially
		if err := bufanalysis.PrintFileAnnotations(container.Stderr(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	return internal.NewBufcliImageWriter(
		container.Logger(),
	).PutImage(
		ctx,
		container,
		container.Output,
		env.Image(),
		container.AsFileDescriptorSet,
		container.ExcludeImports,
	)
}

func imageConvert(ctx context.Context, container *container) (retErr error) {
	if container.Output == "" {
		return fmt.Errorf("--%s is required", imageBuildOutputFlagName)
	}
	image, err := internal.NewBufcliEnvReader(
		container.Logger(),
		imageConvertInputFlagName,
		"",
	).GetImage(
		ctx,
		container,
		container.ConvertInput,
		container.Files,
		false,
		container.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	return internal.NewBufcliImageWriter(
		container.Logger(),
	).PutImage(
		ctx,
		container,
		container.Output,
		image,
		container.AsFileDescriptorSet,
		container.ExcludeImports,
	)
}

func checkLint(ctx context.Context, container *container) (retErr error) {
	asJSON, err := internal.IsLintFormatJSON(errorFormatFlagName, container.ErrorFormat)
	if err != nil {
		return err
	}
	asConfigIgnoreYAML, err := internal.IsLintFormatConfigIgnoreYAML(errorFormatFlagName, container.ErrorFormat)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufcliEnvReader(
		container.Logger(),
		checkLintInputFlagName,
		checkLintConfigFlagName,
	).GetEnv(
		ctx,
		container,
		container.Input,
		container.Config,
		container.Files, // we filter checks for files
		false,           // input files must exist
		false,           // we must include source info for linting
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	fileAnnotations, err = internal.NewBuflintHandler(container.Logger()).Check(
		ctx,
		env.Config().Lint,
		bufimage.ImageWithoutImports(env.Image()),
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if asConfigIgnoreYAML {
			if err := buflint.PrintFileAnnotationsLintConfigIgnoreYAML(container.Stdout(), fileAnnotations); err != nil {
				return err
			}
		} else {
			if err := bufanalysis.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
				return err
			}
		}
		return errors.New("")
	}
	return nil
}

func checkBreaking(ctx context.Context, container *container) (retErr error) {
	if container.AgainstInput == "" {
		return fmt.Errorf("--%s is required", checkBreakingAgainstInputFlagName)
	}
	asJSON, err := internal.IsFormatJSON(errorFormatFlagName, container.ErrorFormat)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufcliEnvReader(
		container.Logger(),
		checkBreakingInputFlagName,
		checkBreakingConfigFlagName,
	).GetEnv(
		ctx,
		container,
		container.Input,
		container.Config,
		container.Files, // we filter checks for files
		false,           // files specified must exist on the main input
		false,           // we must include source info for this side of the check
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	image := env.Image()
	if container.ExcludeImports {
		image = bufimage.ImageWithoutImports(image)
	}

	// TODO: this doesn't actually work because we're using the same file paths for both sides
	// if the roots change, then we're torched
	externalFilePaths := container.Files
	if container.LimitToInputFiles {
		files := image.Files()
		// we know that the file descriptors have unique names from validation
		externalFilePaths = make([]string, len(files))
		for i, file := range files {
			externalFilePaths[i] = file.ExternalFilePath()
		}
	}

	againstEnv, fileAnnotations, err := internal.NewBufcliEnvReader(
		container.Logger(),
		checkBreakingAgainstInputFlagName,
		checkBreakingAgainstConfigFlagName,
	).GetEnv(
		ctx,
		container,
		container.AgainstInput,
		container.AgainstConfig,
		externalFilePaths, // we filter checks for files
		true,              // files are allowed to not exist on the against input
		true,              // no need to include source info for against
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	againstImage := againstEnv.Image()
	if container.ExcludeImports {
		againstImage = bufimage.ImageWithoutImports(againstImage)
	}
	fileAnnotations, err = internal.NewBufbreakingHandler(container.Logger()).Check(
		ctx,
		env.Config().Breaking,
		againstImage,
		image,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	return nil
}

func checkLsLintCheckers(ctx context.Context, container *container) (retErr error) {
	asJSON, err := internal.IsFormatJSON(checkLsCheckersFormatFlagName, container.Format)
	if err != nil {
		return err
	}
	var checkers []bufcheck.Checker
	if container.CheckerAll {
		checkers, err = buflint.GetAllCheckers(container.CheckerCategories...)
		if err != nil {
			return err
		}
	} else {
		config, err := internal.NewBufcliEnvReader(
			container.Logger(),
			"",
			checkLsCheckersConfigFlagName,
		).GetConfig(
			ctx,
			container.Config,
		)
		if err != nil {
			return err
		}
		checkers, err = config.Lint.GetCheckers(container.CheckerCategories...)
		if err != nil {
			return err
		}
	}
	return bufcheck.PrintCheckers(container.Stdout(), checkers, asJSON)
}

func checkLsBreakingCheckers(ctx context.Context, container *container) (retErr error) {
	asJSON, err := internal.IsFormatJSON(checkLsCheckersFormatFlagName, container.Format)
	if err != nil {
		return err
	}
	var checkers []bufcheck.Checker
	if container.CheckerAll {
		checkers, err = bufbreaking.GetAllCheckers(container.CheckerCategories...)
		if err != nil {
			return err
		}
	} else {
		config, err := internal.NewBufcliEnvReader(
			container.Logger(),
			"",
			checkLsCheckersConfigFlagName,
		).GetConfig(
			ctx,
			container.Config,
		)
		if err != nil {
			return err
		}
		checkers, err = config.Breaking.GetCheckers(container.CheckerCategories...)
		if err != nil {
			return err
		}
	}
	return bufcheck.PrintCheckers(container.Stdout(), checkers, asJSON)
}

func lsFiles(ctx context.Context, container *container) (retErr error) {
	fileRefs, err := internal.NewBufcliEnvReader(
		container.Logger(),
		lsFilesInputFlagName,
		lsFilesConfigFlagName,
	).ListFiles(
		ctx,
		container,
		container.Input,
		container.Config,
	)
	if err != nil {
		return err
	}
	for _, fileRef := range fileRefs {
		if _, err := fmt.Fprintln(container.Stdout(), fileRef.ExternalFilePath()); err != nil {
			return err
		}
	}
	return nil
}
