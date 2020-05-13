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

package bufbreaking

import (
	"context"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/proto/protosrc"
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
	previousImage *imagev1beta1.Image,
	image *imagev1beta1.Image,
) ([]*filev1beta1.FileAnnotation, error) {
	previousFiles, err := protosrc.NewFilesUnstable(ctx, previousImage.GetFile()...)
	if err != nil {
		return nil, err
	}
	files, err := protosrc.NewFilesUnstable(ctx, image.GetFile()...)
	if err != nil {
		return nil, err
	}
	return h.breakingRunner.Check(ctx, breakingConfig, previousFiles, files)
}
