package bufbreaking

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
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

func (r *runner) Check(ctx context.Context, config *Config, previousFiles []protodesc.File, files []protodesc.File) ([]*filev1beta1.FileAnnotation, error) {
	return r.delegate.Check(ctx, configToInternalConfig(config), previousFiles, files)
}
