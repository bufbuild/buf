// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufplugin

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/bufplugin-go/check"
	"github.com/bufbuild/pluginrpc-go"
	"go.uber.org/zap"
)

type checkClient struct {
	logger *zap.Logger
	client check.Client
}

func newCheckClient(
	logger *zap.Logger,
	pluginrpcClient pluginrpc.Client,
) *checkClient {
	return &checkClient{
		logger: logger,
		client: check.NewClient(pluginrpcClient),
	}
}

func (c *checkClient) Check(
	ctx context.Context,
	image bufimage.Image,
	againstImage bufimage.Image,
) error {
	files, err := check.FilesForProtoFiles(imageToProtoFiles(image))
	if err != nil {
		return err
	}
	var againstFiles []check.File
	if againstImage != nil {
		againstFiles, err = check.FilesForProtoFiles(imageToProtoFiles(againstImage))
		if err != nil {
			return err
		}
	}
	request, err := check.NewRequest(files, check.WithAgainstFiles(againstFiles))
	if err != nil {
		return err
	}
	response, err := c.client.Check(ctx, request)
	if err != nil {
		return err
	}
	if annotations := response.Annotations(); len(annotations) > 0 {
		return bufanalysis.NewFileAnnotationSet(annotationsToFileAnnotations(annotations)...)
	}
	return nil
}
