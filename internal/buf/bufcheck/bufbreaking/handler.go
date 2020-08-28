// Copyright 2020 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimageutil"
	"github.com/bufbuild/buf/internal/pkg/protosource"
	"go.uber.org/zap"
)

type handler struct {
	logger *zap.Logger
	runner *internal.Runner
}

func newHandler(
	logger *zap.Logger,
) *handler {
	return &handler{
		logger: logger,
		runner: internal.NewRunner(logger, ""),
	}
}

func (h *handler) Check(
	ctx context.Context,
	config *Config,
	previousImage bufimage.Image,
	image bufimage.Image,
) ([]bufanalysis.FileAnnotation, error) {
	previousFiles, err := protosource.NewFilesUnstable(ctx, bufimageutil.NewInputFiles(previousImage.Files())...)
	if err != nil {
		return nil, err
	}
	files, err := protosource.NewFilesUnstable(ctx, bufimageutil.NewInputFiles(image.Files())...)
	if err != nil {
		return nil, err
	}
	return h.runner.Check(ctx, configToInternalConfig(config), previousFiles, files)
}
