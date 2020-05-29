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

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufsrc"
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
	previousImage bufimage.Image,
	image bufimage.Image,
) ([]bufanalysis.FileAnnotation, error) {
	previousFiles, err := bufsrc.NewFilesUnstable(ctx, previousImage.Files()...)
	if err != nil {
		return nil, err
	}
	files, err := bufsrc.NewFilesUnstable(ctx, image.Files()...)
	if err != nil {
		return nil, err
	}
	return h.breakingRunner.Check(ctx, breakingConfig, previousFiles, files)
}
