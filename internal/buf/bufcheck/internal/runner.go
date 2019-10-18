package internal

import (
	"context"

	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/logutil"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// Runner is a runner.
type Runner struct {
	logger *zap.Logger
}

// NewRunner returns a new Runner.
func NewRunner(logger *zap.Logger) *Runner {
	return &Runner{
		logger: logger,
	}
}

// Check runs the Checkers.
func (r *Runner) Check(ctx context.Context, config *Config, previousFiles []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
	checkers := config.Checkers
	if len(checkers) == 0 {
		return nil, nil
	}
	defer logutil.Defer(r.logger, "check", zap.Int("num_files", len(files)), zap.Int("num_checkers", len(checkers)))()

	var annotations []*analysis.Annotation
	var err error
	resultC := make(chan *result, len(checkers))
	for _, checker := range checkers {
		checker := checker
		go func() {
			select {
			case <-ctx.Done():
				iErr := ctx.Err()
				if iErr == context.DeadlineExceeded {
					iErr = errs.NewUserError("timeout")
				}
				resultC <- newResult(nil, iErr)
			default:
			}
			iAnnotations, iErr := checker.check(previousFiles, files)
			resultC <- newResult(iAnnotations, iErr)
		}()
	}
	for i := 0; i < len(checkers); i++ {
		result := <-resultC
		annotations = append(annotations, result.Annotations...)
		err = multierr.Append(err, result.Err)
	}
	if err != nil {
		return nil, err
	}

	if len(config.IgnoreRootPaths) == 0 && len(config.IgnoreIDToRootPaths) == 0 {
		analysis.SortAnnotations(annotations)
		return annotations, nil
	}

	filteredAnnotations := make([]*analysis.Annotation, 0, len(annotations))
	for _, annotation := range annotations {
		if !shouldIgnoreAnnotation(annotation, config.IgnoreRootPaths, config.IgnoreIDToRootPaths) {
			filteredAnnotations = append(filteredAnnotations, annotation)
		}
	}
	analysis.SortAnnotations(filteredAnnotations)
	return filteredAnnotations, nil
}

func shouldIgnoreAnnotation(annotation *analysis.Annotation, ignoreAllRootPaths map[string]struct{}, ignoreIDToRootPaths map[string]map[string]struct{}) bool {
	if annotation.Filename == "" {
		return false
	}
	if storagepath.MapContainsMatch(ignoreAllRootPaths, annotation.Filename) {
		return true
	}
	if annotation.Type == "" {
		return false
	}
	ignoreRootPaths, ok := ignoreIDToRootPaths[annotation.Type]
	if !ok {
		return false
	}
	return storagepath.MapContainsMatch(ignoreRootPaths, annotation.Filename)
}

type result struct {
	Annotations []*analysis.Annotation
	Err         error
}

func newResult(annotations []*analysis.Annotation, err error) *result {
	return &result{
		Annotations: annotations,
		Err:         err,
	}
}
