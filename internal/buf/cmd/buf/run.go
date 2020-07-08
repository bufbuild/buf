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
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
)

func imageBuild(ctx context.Context, container applog.Container, flags *flags) (retErr error) {
	if flags.Output == "" {
		return fmt.Errorf("--%s is required", imageBuildOutputFlagName)
	}
	env, fileAnnotations, err := internal.NewBufwireEnvReader(
		container.Logger(),
		imageBuildInputFlagName,
		imageBuildConfigFlagName,
		// must be source only
	).GetSourceEnv(
		ctx,
		container,
		flags.Input,
		flags.Config,
		flags.Files,
		false,
		flags.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdout potentially
		if err := bufanalysis.PrintFileAnnotations(
			container.Stderr(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		// app works on the concept that an error results in a non-zero exit code
		// we already printed the messages with PrintFileAnnotations so we do
		// not want to print any additional error message
		// we could put the FileAnnotations in this error, but in general with
		// linting/breaking change detection we actually print them to stdout
		// so doing this here is consistent with lint/breaking change detection
		return errors.New("")
	}
	return internal.NewBufwireImageWriter(
		container.Logger(),
	).PutImage(
		ctx,
		container,
		flags.Output,
		env.Image(),
		flags.AsFileDescriptorSet,
		flags.ExcludeImports,
	)
}

func imageConvert(ctx context.Context, container applog.Container, flags *flags) (retErr error) {
	internal.WarnExperimental(container)
	if flags.Output == "" {
		return fmt.Errorf("--%s is required", imageBuildOutputFlagName)
	}
	image, err := internal.NewBufwireImageReader(
		container.Logger(),
		imageConvertInputFlagName,
	).GetImage(
		ctx,
		container,
		flags.ConvertInput,
		flags.Files,
		false,
		flags.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	return internal.NewBufwireImageWriter(
		container.Logger(),
	).PutImage(
		ctx,
		container,
		flags.Output,
		image,
		flags.AsFileDescriptorSet,
		flags.ExcludeImports,
	)
}

func checkLint(ctx context.Context, container applog.Container, flags *flags) (retErr error) {
	env, fileAnnotations, err := internal.NewBufwireEnvReader(
		container.Logger(),
		checkLintInputFlagName,
		checkLintConfigFlagName,
	).GetEnv(
		ctx,
		container,
		flags.Input,
		flags.Config,
		flags.Files, // we filter checks for files
		false,       // input files must exist
		false,       // we must include source info for linting
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		formatString := flags.ErrorFormat
		if formatString == "config-ignore-yaml" {
			formatString = "text"
		}
		if err := bufanalysis.PrintFileAnnotations(container.Stdout(), fileAnnotations, formatString); err != nil {
			return err
		}
		return errors.New("")
	}
	fileAnnotations, err = internal.NewBuflintHandler(container.Logger()).Check(
		ctx,
		env.Config().Lint,
		bufcore.ImageWithoutImports(env.Image()),
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := buflint.PrintFileAnnotations(
			container.Stdout(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return errors.New("")
	}
	return nil
}

func checkBreaking(ctx context.Context, container applog.Container, flags *flags) (retErr error) {
	if flags.AgainstInput == "" {
		return fmt.Errorf("--%s is required", checkBreakingAgainstInputFlagName)
	}
	env, fileAnnotations, err := internal.NewBufwireEnvReader(
		container.Logger(),
		checkBreakingInputFlagName,
		checkBreakingConfigFlagName,
	).GetEnv(
		ctx,
		container,
		flags.Input,
		flags.Config,
		flags.Files, // we filter checks for files
		false,       // files specified must exist on the main input
		false,       // we must include source info for this side of the check
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(
			container.Stdout(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return errors.New("")
	}
	image := env.Image()
	if flags.ExcludeImports {
		image = bufcore.ImageWithoutImports(image)
	}

	// TODO: this doesn't actually work because we're using the same file paths for both sides
	// if the roots change, then we're torched
	externalPaths := flags.Files
	if flags.LimitToInputFiles {
		files := image.Files()
		// we know that the file descriptors have unique names from validation
		externalPaths = make([]string, len(files))
		for i, file := range files {
			externalPaths[i] = file.ExternalPath()
		}
	}

	againstEnv, fileAnnotations, err := internal.NewBufwireEnvReader(
		container.Logger(),
		checkBreakingAgainstInputFlagName,
		checkBreakingAgainstConfigFlagName,
	).GetEnv(
		ctx,
		container,
		flags.AgainstInput,
		flags.AgainstConfig,
		externalPaths, // we filter checks for files
		true,          // files are allowed to not exist on the against input
		true,          // no need to include source info for against
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(
			container.Stdout(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return errors.New("")
	}
	againstImage := againstEnv.Image()
	if flags.ExcludeImports {
		againstImage = bufcore.ImageWithoutImports(againstImage)
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
		if err := bufanalysis.PrintFileAnnotations(
			container.Stdout(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return errors.New("")
	}
	return nil
}

func checkLsLintCheckers(ctx context.Context, container applog.Container, flags *flags) (retErr error) {
	var checkers []bufcheck.Checker
	var err error
	if flags.CheckerAll {
		checkers, err = buflint.GetAllCheckers(flags.CheckerCategories...)
		if err != nil {
			return err
		}
	} else {
		config, err := internal.NewBufwireEnvReader(
			container.Logger(),
			"",
			checkLsCheckersConfigFlagName,
		).GetConfig(
			ctx,
			flags.Config,
		)
		if err != nil {
			return err
		}
		checkers, err = config.Lint.GetCheckers(flags.CheckerCategories...)
		if err != nil {
			return err
		}
	}
	return bufcheck.PrintCheckers(
		container.Stdout(),
		checkers,
		flags.Format,
	)
}

func checkLsBreakingCheckers(ctx context.Context, container applog.Container, flags *flags) (retErr error) {
	var checkers []bufcheck.Checker
	var err error
	if flags.CheckerAll {
		checkers, err = bufbreaking.GetAllCheckers(flags.CheckerCategories...)
		if err != nil {
			return err
		}
	} else {
		config, err := internal.NewBufwireEnvReader(
			container.Logger(),
			"",
			checkLsCheckersConfigFlagName,
		).GetConfig(
			ctx,
			flags.Config,
		)
		if err != nil {
			return err
		}
		checkers, err = config.Breaking.GetCheckers(flags.CheckerCategories...)
		if err != nil {
			return err
		}
	}
	return bufcheck.PrintCheckers(
		container.Stdout(),
		checkers,
		flags.Format,
	)
}
