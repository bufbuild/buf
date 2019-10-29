package buflint

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"go.uber.org/zap"
)

type handler struct {
	logger     *zap.Logger
	lintRunner Runner
}

func newHandler(
	logger *zap.Logger,
	lintRunner Runner,
) *handler {
	return &handler{
		logger:     logger.Named("buflint"),
		lintRunner: lintRunner,
	}
}

func (h *handler) LintCheck(
	ctx context.Context,
	lintConfig *Config,
	image bufpb.Image,
) ([]*analysis.Annotation, error) {
	files, err := protodesc.NewFiles(ctx, image.GetFile()...)
	if err != nil {
		return nil, err
	}
	return h.lintRunner.Check(ctx, lintConfig, files)
}
