package buflint

import (
	"context"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
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
	image *imagev1beta1.Image,
) ([]*filev1beta1.FileAnnotation, error) {
	files, err := protodesc.NewFilesUnstable(ctx, image.GetFile()...)
	if err != nil {
		return nil, err
	}
	return h.lintRunner.Check(ctx, lintConfig, files)
}
