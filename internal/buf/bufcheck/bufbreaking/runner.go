package bufbreaking

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"go.uber.org/zap"
)

type runner struct {
	delegate *internal.Runner
}

func newRunner(logger *zap.Logger) *runner {
	return &runner{
		delegate: internal.NewRunner(logger.Named("breaking")),
	}
}

func (r *runner) Check(ctx context.Context, config *Config, previousFiles []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
	return r.delegate.Check(ctx, configToInternalConfig(config), previousFiles, files)
}
