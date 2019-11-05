package buf

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/bytepool"
	"github.com/bufbuild/buf/internal/pkg/cli"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"go.uber.org/zap"
)

func imageBuild(
	ctx context.Context,
	execEnv *cli.ExecEnv,
	flags *Flags,
	logger *zap.Logger,
	segList *bytepool.SegList,
) (retErr error) {
	if flags.Output == "" {
		return errs.NewInvalidArgumentf("--%s is required", imageBuildOutputFlagName)
	}
	asJSON, err := internal.IsFormatJSON(errorFormatFlagName, flags.ErrorFormat)
	if err != nil {
		return err
	}
	env, annotations, err := internal.NewBufosEnvReader(
		logger,
		segList,
		imageBuildInputFlagName,
		imageBuildConfigFlagName,
		// must be source only
	).ReadSourceEnv(
		ctx,
		execEnv.Stdin,
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
	if len(annotations) > 0 {
		// stderr since we do output to stdout potentially
		if err := analysis.PrintAnnotations(execEnv.Stderr, annotations, asJSON); err != nil {
			return err
		}
		return errs.NewInternal("")
	}
	return internal.NewBufosImageWriter(
		logger,
		imageBuildOutputFlagName,
	).WriteImage(
		ctx,
		execEnv.Stdout,
		flags.Output,
		flags.AsFileDescriptorSet,
		env.Image,
	)
}

func checkLint(
	ctx context.Context,
	execEnv *cli.ExecEnv,
	flags *Flags,
	logger *zap.Logger,
	segList *bytepool.SegList,
) (retErr error) {
	asJSON, err := internal.IsLintFormatJSON(errorFormatFlagName, flags.ErrorFormat)
	if err != nil {
		return err
	}
	asConfigIgnoreYAML, err := internal.IsLintFormatConfigIgnoreYAML(errorFormatFlagName, flags.ErrorFormat)
	if err != nil {
		return err
	}
	env, annotations, err := internal.NewBufosEnvReader(
		logger,
		segList,
		checkLintInputFlagName,
		checkLintConfigFlagName,
	).ReadEnv(
		ctx,
		execEnv.Stdin,
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
	if len(annotations) > 0 {
		if err := analysis.PrintAnnotations(execEnv.Stdout, annotations, asJSON); err != nil {
			return err
		}
		return errs.NewInternal("")
	}
	annotations, err = internal.NewBuflintHandler(logger).LintCheck(
		ctx,
		env.Config.Lint,
		env.Image,
	)
	if err != nil {
		return err
	}
	if len(annotations) > 0 {
		if asConfigIgnoreYAML {
			if err := bufconfig.PrintAnnotationsLintConfigIgnoreYAML(execEnv.Stdout, annotations); err != nil {
				return err
			}
		} else {
			if err := bufbuild.FixAnnotationFilenames(env.Resolver, annotations); err != nil {
				return err
			}
			if err := analysis.PrintAnnotations(execEnv.Stdout, annotations, asJSON); err != nil {
				return err
			}
		}
		return errs.NewInternal("")
	}
	return nil
}

func checkBreaking(
	ctx context.Context,
	execEnv *cli.ExecEnv,
	flags *Flags,
	logger *zap.Logger,
	segList *bytepool.SegList,
) (retErr error) {
	if flags.AgainstInput == "" {
		return errs.NewInvalidArgumentf("--%s is required", checkBreakingAgainstInputFlagName)
	}
	asJSON, err := internal.IsFormatJSON(errorFormatFlagName, flags.ErrorFormat)
	if err != nil {
		return err
	}
	env, annotations, err := internal.NewBufosEnvReader(
		logger,
		segList,
		checkBreakingInputFlagName,
		checkBreakingConfigFlagName,
	).ReadEnv(
		ctx,
		execEnv.Stdin,
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
	if len(annotations) > 0 {
		if err := analysis.PrintAnnotations(execEnv.Stdout, annotations, asJSON); err != nil {
			return err
		}
		return errs.NewInternal("")
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

	againstEnv, annotations, err := internal.NewBufosEnvReader(
		logger,
		segList,
		checkBreakingAgainstInputFlagName,
		checkBreakingAgainstConfigFlagName,
	).ReadEnv(
		ctx,
		execEnv.Stdin,
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
	if len(annotations) > 0 {
		// TODO: formalize this somewhere
		for _, annotation := range annotations {
			if annotation.Filename != "" {
				annotation.Filename = annotation.Filename + "@against"
			}
		}
		if err := analysis.PrintAnnotations(execEnv.Stdout, annotations, asJSON); err != nil {
			return err
		}
		return errs.NewInternal("")
	}
	annotations, err = internal.NewBufbreakingHandler(logger).BreakingCheck(
		ctx,
		env.Config.Breaking,
		againstEnv.Image,
		env.Image,
	)
	if err != nil {
		return err
	}
	if len(annotations) > 0 {
		if err := bufbuild.FixAnnotationFilenames(env.Resolver, annotations); err != nil {
			return err
		}
		if err := analysis.PrintAnnotations(execEnv.Stdout, annotations, asJSON); err != nil {
			return err
		}
		return errs.NewInternal("")
	}
	return nil
}

func checkLsLintCheckers(
	ctx context.Context,
	execEnv *cli.ExecEnv,
	flags *Flags,
	logger *zap.Logger,
	segList *bytepool.SegList,
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
			segList,
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
	return bufcheck.PrintCheckers(execEnv.Stdout, checkers, asJSON)
}

func checkLsBreakingCheckers(
	ctx context.Context,
	execEnv *cli.ExecEnv,
	flags *Flags,
	logger *zap.Logger,
	segList *bytepool.SegList,
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
			segList,
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
	return bufcheck.PrintCheckers(execEnv.Stdout, checkers, asJSON)
}

func lsFiles(
	ctx context.Context,
	execEnv *cli.ExecEnv,
	flags *Flags,
	logger *zap.Logger,
	segList *bytepool.SegList,
) (retErr error) {
	filePaths, err := internal.NewBufosEnvReader(
		logger,
		segList,
		lsFilesInputFlagName,
		lsFilesConfigFlagName,
	).ListFiles(
		ctx,
		execEnv.Stdin,
		flags.Input,
		flags.Config,
	)
	if err != nil {
		return err
	}
	for _, filePath := range filePaths {
		if _, err := fmt.Fprintln(execEnv.Stdout, filePath); err != nil {
			return err
		}
	}
	return nil
}
