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

package protoc

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"github.com/bufbuild/buf/internal/pkg/app/appproto/appprotoexec"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/thread"
	"go.uber.org/zap"
)

type pluginInfo struct {
	// Required
	Out string
	// optional
	Opt []string
	// optional
	Path string
}

func newPluginInfo() *pluginInfo {
	return &pluginInfo{}
}

func executePlugin(
	ctx context.Context,
	logger *zap.Logger,
	container app.EnvStderrContainer,
	images []bufimage.Image,
	pluginName string,
	pluginInfo *pluginInfo,
) error {
	switch filepath.Ext(pluginInfo.Out) {
	case ".jar":
		return fmt.Errorf("jar output not supported but is coming soon: %q", pluginInfo.Out)
	case ".zip":
		return fmt.Errorf("zip output not supported but is coming soon: %q", pluginInfo.Out)
	}
	readWriteBucket, err := storageos.NewReadWriteBucket(pluginInfo.Out)
	if err != nil {
		return err
	}
	var handlerOptions []appprotoexec.HandlerOption
	if pluginInfo.Path != "" {
		handlerOptions = append(handlerOptions, appprotoexec.HandlerWithPluginPath(pluginInfo.Path))
	}
	handler, err := appprotoexec.NewHandler(logger, pluginName, handlerOptions...)
	if err != nil {
		return err
	}
	executor := appproto.NewExecutor(logger, handler)
	jobs := make([]func() error, len(images))
	for i, image := range images {
		image := image
		jobs[i] = func() error {
			if err := executor.Execute(
				ctx,
				container,
				readWriteBucket,
				bufimage.ImageToCodeGeneratorRequest(
					image,
					strings.Join(pluginInfo.Opt, ","),
				),
			); err != nil {
				return fmt.Errorf("--%s_out: %v", pluginName, err)
			}
			return nil
		}
	}
	return thread.Parallelize(jobs...)
}
