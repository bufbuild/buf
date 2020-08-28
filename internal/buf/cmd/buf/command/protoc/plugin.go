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
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
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
	handler, err := appprotoexec.NewHandler(logger, pluginName, "", pluginInfo.Path)
	if err != nil {
		return err
	}
	jobs := make([]func() error, len(images))
	for i, image := range images {
		image := image
		jobs[i] = func() error {
			return executePluginForImage(
				ctx,
				container,
				image,
				pluginName,
				pluginInfo,
				handler,
			)
		}
	}
	return thread.Parallelize(jobs...)
}

func executePluginForImage(
	ctx context.Context,
	container app.EnvStderrContainer,
	image bufimage.Image,
	pluginName string,
	pluginInfo *pluginInfo,
	handler appproto.Handler,
) error {
	request := bufimage.ImageToCodeGeneratorRequest(image, strings.Join(pluginInfo.Opt, ","))
	response, err := appproto.Execute(ctx, container, handler, request)
	if err != nil {
		return err
	}
	if errString := response.GetError(); errString != "" {
		return fmt.Errorf("--%s_out: %s", pluginName, errString)
	}
	if err := writeResponseFiles(ctx, response.File, pluginInfo.Out); err != nil {
		return fmt.Errorf("--%s_out: %v", pluginName, err)
	}
	return nil
}

func writeResponseFiles(
	ctx context.Context,
	files []*pluginpb.CodeGeneratorResponse_File,
	out string,
) error {
	switch filepath.Ext(out) {
	case ".jar":
		return fmt.Errorf("jar output not supported but is coming soon: %q", out)
	case ".zip":
		return fmt.Errorf("zip output not supported but is coming soon: %q", out)
	}
	readWriteBucket, err := storageos.NewReadWriteBucket(out)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.GetInsertionPoint() != "" {
			return fmt.Errorf("insertion points not supported but are coming soon: %s", file.GetName())
		}
		data := []byte(file.GetContent())
		writeObjectCloser, err := readWriteBucket.Put(ctx, file.GetName(), uint32(len(data)))
		if err != nil {
			return err
		}
		_, err = writeObjectCloser.Write(data)
		err = multierr.Append(err, writeObjectCloser.Close())
		if err != nil {
			return err
		}
	}
	return nil
}
