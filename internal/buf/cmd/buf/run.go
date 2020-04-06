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
	"github.com/bufbuild/buf/internal/pkg/cli/clienv"
	"go.uber.org/zap"
)

func imageBuild(
	ctx context.Context,
	cliEnv clienv.Env,
	flags *Flags,
	logger *zap.Logger,
) (retErr error) {
	if flags.Output == "" {
		return fmt.Errorf("--%s is required", imageBuildOutputFlagName)
	}
	asJSON, err := internal.IsFormatJSON(errorFormatFlagName, flags.ErrorFormat)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufosEnvReader(
		logger,
		imageBuildInputFlagName,
		imageBuildConfigFlagName,
		// must be source only
	).ReadSourceEnv(
		ctx,
		cliEnv.Stdin(),
		cliEnv.Getenv,
		flags.Input,
		flags.Config,
		nil,   // we do not filter files for images
		false, // this is ignored since we do not specify specific files
		!flags.ExcludeImports,
		!flags.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdout potentially
		if err := extfile.PrintFileAnnotations(cliEnv.Stderr(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	return internal.NewBufosImageWriter(
		logger,
		imageBuildOutputFlagName,
	).WriteImage(
		ctx,
		cliEnv.Stdout(),
		flags.Output,
		flags.AsFileDescriptorSet,
		env.Image,
	)
}

func checkLint(
	ctx context.Context,
	cliEnv clienv.Env,
	flags *Flags,
	logger *zap.Logger,
) (retErr error) {
	asJSON, err := internal.IsLintFormatJSON(errorFormatFlagName, flags.ErrorFormat)
	if err != nil {
		return err
	}
	asConfigIgnoreYAML, err := internal.IsLintFormatConfigIgnoreYAML(errorFormatFlagName, flags.ErrorFormat)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufosEnvReader(
		logger,
		checkLintInputFlagName,
		checkLintConfigFlagName,
	).ReadEnv(
		ctx,
		cliEnv.Stdin(),
		cliEnv.Getenv,
		flags.Input,
		flags.Config,
		flags.Files, // we filter checks for files
		false,       // input files must exist
		false,       // do not want to include imports
		true,        // we must include source info for linting
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := extfile.PrintFileAnnotations(cliEnv.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	fileAnnotations, err = internal.NewBuflintHandler(logger).LintCheck(
		ctx,
		env.Config.Lint,
		env.Image,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if asConfigIgnoreYAML {
			if err := bufconfig.PrintFileAnnotationsLintConfigIgnoreYAML(cliEnv.Stdout(), fileAnnotations); err != nil {
				return err
			}
		} else {
			if err := bufbuild.FixFileAnnotationPaths(env.Resolver, fileAnnotations...); err != nil {
				return err
			}
			if err := extfile.PrintFileAnnotations(cliEnv.Stdout(), fileAnnotations, asJSON); err != nil {
				return err
			}
		}
		return errors.New("")
	}
	return nil
}

func checkBreaking(
	ctx context.Context,
	cliEnv clienv.Env,
	flags *Flags,
	logger *zap.Logger,
) (retErr error) {
	if flags.AgainstInput == "" {
		return fmt.Errorf("--%s is required", checkBreakingAgainstInputFlagName)
	}
	asJSON, err := internal.IsFormatJSON(errorFormatFlagName, flags.ErrorFormat)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufosEnvReader(
		logger,
		checkBreakingInputFlagName,
		checkBreakingConfigFlagName,
	).ReadEnv(
		ctx,
		cliEnv.Stdin(),
		cliEnv.Getenv,
		flags.Input,
		flags.Config,
		flags.Files, // we filter checks for files
		false,       // files specified must exist on the main input
		!flags.ExcludeImports,
		true, // we must include source info for this side of the check
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := extfile.PrintFileAnnotations(cliEnv.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}

	files := flags.Files
	if flags.LimitToInputFiles {
		fileDescriptors := env.Image.GetFile()
		// we know that the file descriptors have unique names from validation
		files = make([]string, len(fileDescriptors))
		for i, fileDescriptor := range fileDescriptors {
			// we know that the name is non-empty from validation
			files[i] = fileDescriptor.GetName()
		}
	}

	againstEnv, fileAnnotations, err := internal.NewBufosEnvReader(
		logger,
		checkBreakingAgainstInputFlagName,
		checkBreakingAgainstConfigFlagName,
	).ReadEnv(
		ctx,
		cliEnv.Stdin(),
		cliEnv.Getenv,
		flags.AgainstInput,
		flags.AgainstConfig,
		files, // we filter checks for files
		true,  // files are allowed to not exist on the against input
		!flags.ExcludeImports,
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
		if err := extfile.PrintFileAnnotations(cliEnv.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	fileAnnotations, err = internal.NewBufbreakingHandler(logger).BreakingCheck(
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
		if err := extfile.PrintFileAnnotations(cliEnv.Stdout(), fileAnnotations, asJSON); err != nil {
			return err
		}
		return errors.New("")
	}
	return nil
}

func checkLsLintCheckers(
	ctx context.Context,
	cliEnv clienv.Env,
	flags *Flags,
	logger *zap.Logger,
) (retErr error) {
	asJSON, err := internal.IsFormatJSON(checkLsCheckersFormatFlagName, flags.Format)
	if err != nil {
		return err
	}
	var checkers []bufcheck.Checker
	if flags.CheckerAll {
		checkers, err = buflint.GetAllCheckers(flags.CheckerCategories...)
		if err != nil {
			return err
		}
	} else {
		config, err := internal.NewBufosEnvReader(
			logger,
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
	return bufcheck.PrintCheckers(cliEnv.Stdout(), checkers, asJSON)
}

func checkLsBreakingCheckers(
	ctx context.Context,
	cliEnv clienv.Env,
	flags *Flags,
	logger *zap.Logger,
) (retErr error) {
	asJSON, err := internal.IsFormatJSON(checkLsCheckersFormatFlagName, flags.Format)
	if err != nil {
		return err
	}
	var checkers []bufcheck.Checker
	if flags.CheckerAll {
		checkers, err = bufbreaking.GetAllCheckers(flags.CheckerCategories...)
		if err != nil {
			return err
		}
	} else {
		config, err := internal.NewBufosEnvReader(
			logger,
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
	return bufcheck.PrintCheckers(cliEnv.Stdout(), checkers, asJSON)
}

func lsFiles(
	ctx context.Context,
	cliEnv clienv.Env,
	flags *Flags,
	logger *zap.Logger,
) (retErr error) {
	filePaths, err := internal.NewBufosEnvReader(
		logger,
		lsFilesInputFlagName,
		lsFilesConfigFlagName,
	).ListFiles(
		ctx,
		cliEnv.Stdin(),
		cliEnv.Getenv,
		flags.Input,
		flags.Config,
	)
	if err != nil {
		return err
	}
	for _, filePath := range filePaths {
		if _, err := fmt.Fprintln(cliEnv.Stdout(), filePath); err != nil {
			return err
		}
	}
	return nil
}
