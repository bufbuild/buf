// Copyright 2020 Buf Technologies Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/buf/ext/extfile"
)

func imageBuild(ctx context.Context, container *container) (retErr error) {
	if container.Output == "" {
		return fmt.Errorf("--%s is required", imageBuildOutputFlagName)
	}
	asJSON, err := internal.IsFormatJSON(errorFormatFlagName, container.ErrorFormat)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufosEnvReader(
		container.Logger(),
		imageBuildInputFlagName,
		imageBuildConfigFlagName,
		container.ExperimentalGitClone,
		// must be source only
	).ReadSourceEnv(
		ctx,
		container,
		container.Input,
		container.Config,
		nil,   // we do not filter files for images
		false, // this is ignored since we do not specify specific files
		!container.ExcludeImports,
		!container.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdout potentially
		if err := extfile.PrintFileAnnotations(container.Stderr(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	return internal.NewBufosImageWriter(
		container.Logger(),
		imageBuildOutputFlagName,
	).WriteImage(
		ctx,
		container,
		container.Output,
		container.AsFileDescriptorSet,
		env.Image,
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
	env, fileAnnotations, err := internal.NewBufosEnvReader(
		container.Logger(),
		checkLintInputFlagName,
		checkLintConfigFlagName,
		container.ExperimentalGitClone,
	).ReadEnv(
		ctx,
		container,
		container.Input,
		container.Config,
		container.Files, // we filter checks for files
		false,           // input files must exist
		false,           // do not want to include imports
		true,            // we must include source info for linting
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := extfile.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	fileAnnotations, err = internal.NewBuflintHandler(container.Logger()).LintCheck(
		ctx,
		env.Config.Lint,
		env.Image,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if asConfigIgnoreYAML {
			if err := bufconfig.PrintFileAnnotationsLintConfigIgnoreYAML(container.Stdout(), fileAnnotations); err != nil {
				return err
			}
		} else {
			if err := bufbuild.FixFileAnnotationPaths(env.Resolver, fileAnnotations...); err != nil {
				return err
			}
			if err := extfile.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
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
	env, fileAnnotations, err := internal.NewBufosEnvReader(
		container.Logger(),
		checkBreakingInputFlagName,
		checkBreakingConfigFlagName,
		container.ExperimentalGitClone,
	).ReadEnv(
		ctx,
		container,
		container.Input,
		container.Config,
		container.Files, // we filter checks for files
		false,           // files specified must exist on the main input
		!container.ExcludeImports,
		true, // we must include source info for this side of the check
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := extfile.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}

	files := container.Files
	if container.LimitToInputFiles {
		fileDescriptors := env.Image.GetFile()
		// we know that the file descriptors have unique names from validation
		files = make([]string, len(fileDescriptors))
		for i, fileDescriptor := range fileDescriptors {
			// we know that the name is non-empty from validation
			files[i] = fileDescriptor.GetName()
		}
	}

	againstEnv, fileAnnotations, err := internal.NewBufosEnvReader(
		container.Logger(),
		checkBreakingAgainstInputFlagName,
		checkBreakingAgainstConfigFlagName,
		container.ExperimentalGitClone,
	).ReadEnv(
		ctx,
		container,
		container.AgainstInput,
		container.AgainstConfig,
		files, // we filter checks for files
		true,  // files are allowed to not exist on the against input
		!container.ExcludeImports,
		false, // no need to include source info for against
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		// TODO: formalize this somewhere
		for _, fileAnnotation := range fileAnnotations {
			if fileAnnotation.Path != "" {
				fileAnnotation.Path = fileAnnotation.Path + "@against"
			}
		}
		if err := extfile.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	fileAnnotations, err = internal.NewBufbreakingHandler(container.Logger()).BreakingCheck(
		ctx,
		env.Config.Breaking,
		againstEnv.Image,
		env.Image,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufbuild.FixFileAnnotationPaths(env.Resolver, fileAnnotations...); err != nil {
			return err
		}
		if err := extfile.PrintFileAnnotations(container.Stdout(), fileAnnotations, asJSON); err != nil {
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
		config, err := internal.NewBufosEnvReader(
			container.Logger(),
			"",
			checkLsCheckersConfigFlagName,
			container.ExperimentalGitClone,
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
		config, err := internal.NewBufosEnvReader(
			container.Logger(),
			"",
			checkLsCheckersConfigFlagName,
			container.ExperimentalGitClone,
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
	filePaths, err := internal.NewBufosEnvReader(
		container.Logger(),
		lsFilesInputFlagName,
		lsFilesConfigFlagName,
		container.ExperimentalGitClone,
	).ListFiles(
		ctx,
		container,
		container.Input,
		container.Config,
	)
	if err != nil {
		return err
	}
	for _, filePath := range filePaths {
		if _, err := fmt.Fprintln(container.Stdout(), filePath); err != nil {
			return err
		}
	}
	return nil
}
