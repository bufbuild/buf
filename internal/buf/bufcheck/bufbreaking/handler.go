package bufbreaking

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"go.uber.org/zap"
)

type handler struct {
	logger         *zap.Logger
	breakingRunner Runner
}

func newHandler(
	logger *zap.Logger,
	breakingRunner Runner,
) *handler {
	return &handler{
		logger:         logger.Named("bufbreaking"),
		breakingRunner: breakingRunner,
	}
}

func (h *handler) BreakingCheck(
	ctx context.Context,
	breakingConfig *Config,
	previousImage bufpb.Image,
	image bufpb.Image,
) ([]*analysis.Annotation, error) {
	previousFiles, err := protodesc.NewFiles(previousImage.GetFile()...)
	if err != nil {
		return nil, err
	}
	files, err := protodesc.NewFiles(image.GetFile()...)
	if err != nil {
		return nil, err
	}
	return h.breakingRunner.Check(ctx, breakingConfig, previousFiles, files)
}
