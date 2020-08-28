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

package bufwire

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/image/v1"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type imageReader struct {
	logger      *zap.Logger
	fetchReader buffetch.ImageReader
}

func newImageReader(
	logger *zap.Logger,
	fetchReader buffetch.ImageReader,
) *imageReader {
	return &imageReader{
		logger:      logger.Named("bufwire"),
		fetchReader: fetchReader,
	}
}

func (i *imageReader) GetImage(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef buffetch.ImageRef,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ bufimage.Image, retErr error) {
	ctx, span := trace.StartSpan(ctx, "get_image")
	defer span.End()
	readCloser, err := i.fetchReader.GetImageFile(ctx, container, imageRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	data, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}
	protoImage := &imagev1.Image{}
	switch imageEncoding := imageRef.ImageEncoding(); imageEncoding {
	// we have to double parse due to custom options
	// See https://github.com/golang/protobuf/issues/1123
	// TODO: revisit
	case buffetch.ImageEncodingBin:
		firstProtoImage := &imagev1.Image{}
		_, span := trace.StartSpan(ctx, "first_wire_unmarshal")
		if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, firstProtoImage); err != nil {
			return nil, fmt.Errorf("could not unmarshal Image: %v", err)
		}
		span.End()
		_, span = trace.StartSpan(ctx, "new_resolver")
		// TODO right now, NewResolver sets AllowUnresolvable to true all the time
		// we want to make this into a check, and we verify if we need this for the individual command
		resolver, err := protoencoding.NewResolver(
			firstProtoImage.File...,
		)
		if err != nil {
			return nil, err
		}
		span.End()
		_, span = trace.StartSpan(ctx, "second_wire_unmarshal")
		if err := protoencoding.NewWireUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			return nil, fmt.Errorf("could not unmarshal Image: %v", err)
		}
		span.End()
	case buffetch.ImageEncodingJSON:
		firstProtoImage := &imagev1.Image{}
		_, span := trace.StartSpan(ctx, "first_json_unmarshal")
		if err := protoencoding.NewJSONUnmarshaler(nil).Unmarshal(data, firstProtoImage); err != nil {
			return nil, fmt.Errorf("could not unmarshal Image: %v", err)
		}
		// TODO right now, NewResolver sets AllowUnresolvable to true all the time
		// we want to make this into a check, and we verify if we need this for the individual command
		span.End()
		_, span = trace.StartSpan(ctx, "new_resolver")
		resolver, err := protoencoding.NewResolver(
			firstProtoImage.File...,
		)
		if err != nil {
			return nil, err
		}
		span.End()
		_, span = trace.StartSpan(ctx, "second_json_unmarshal")
		if err := protoencoding.NewJSONUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			return nil, fmt.Errorf("could not unmarshal Image: %v", err)
		}
		span.End()
	default:
		return nil, fmt.Errorf("unknown image encoding: %v", imageEncoding)
	}
	if excludeSourceCodeInfo {
		for _, fileDescriptorProto := range protoImage.File {
			fileDescriptorProto.SourceCodeInfo = nil
		}
	}
	image, err := bufimage.NewImageForProto(protoImage)
	if err != nil {
		return nil, err
	}
	if len(externalFilePaths) == 0 {
		return image, nil
	}
	imagePaths := make([]string, len(externalFilePaths))
	for i, externalFilePath := range externalFilePaths {
		imagePath, err := imageRef.PathForExternalPath(externalFilePath)
		if err != nil {
			return nil, err
		}
		imagePaths[i] = imagePath
	}
	if externalFilePathsAllowNotExist {
		// externalFilePaths have to be targetPaths
		// TODO: evaluate this
		return bufimage.ImageWithOnlyPathsAllowNotExist(image, imagePaths)
	}
	return bufimage.ImageWithOnlyPaths(image, imagePaths)
}
