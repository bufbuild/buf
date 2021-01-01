// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufgen

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appproto/appprotoos"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/zap"
)

type generator struct {
	logger              *zap.Logger
	appprotoosGenerator appprotoos.Generator
}

func newGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) *generator {
	return &generator{
		logger:              logger,
		appprotoosGenerator: appprotoos.NewGenerator(logger, storageosProvider),
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStdioContainer,
	config *Config,
	image bufimage.Image,
	options ...GenerateOption,
) error {
	generateOptions := newGenerateOptions()
	for _, option := range options {
		option(generateOptions)
	}
	return g.generate(
		ctx,
		container,
		config,
		image,
		generateOptions.baseOutDirPath,
	)
}

func (g *generator) generate(
	ctx context.Context,
	container app.EnvStdioContainer,
	config *Config,
	image bufimage.Image,
	baseOutDirPath string,
) error {
	// we keep this as a variable so we can cache it if we hit StrategyDirectory
	var imagesByDir []bufimage.Image
	var err error
	for _, pluginConfig := range config.PluginConfigs {
		out := pluginConfig.Out
		if baseOutDirPath != "" && baseOutDirPath != "." {
			out = filepath.Join(baseOutDirPath, out)
		}
		var pluginImages []bufimage.Image
		switch pluginConfig.Strategy {
		case StrategyAll:
			pluginImages = []bufimage.Image{image}
		case StrategyDirectory:
			// if we have not already called this, call it
			if imagesByDir == nil {
				imagesByDir, err = bufimage.ImageByDir(image)
				if err != nil {
					return err
				}
			}
			pluginImages = imagesByDir
		default:
			return fmt.Errorf("unknown strategy: %v", pluginConfig.Strategy)
		}
		if err := g.appprotoosGenerator.Generate(
			ctx,
			container,
			pluginConfig.Name,
			out,
			bufimage.ImagesToCodeGeneratorRequests(pluginImages, pluginConfig.Opt),
			appprotoos.GenerateWithPluginPath(pluginConfig.Path),
			appprotoos.GenerateWithCreateOutDirIfNotExists(),
		); err != nil {
			return fmt.Errorf("plugin %s: %v", pluginConfig.Name, err)
		}
	}
	return nil
}

type generateOptions struct {
	baseOutDirPath string
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
