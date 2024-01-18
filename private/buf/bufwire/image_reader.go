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

package bufwire

import (
	"context"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	loggerName = "bufwire"
	tracerName = "bufbuild/buf"
)

type imageReader struct {
	logger      *zap.Logger
	fetchReader buffetch.MessageReader
	tracer      trace.Tracer
}

func newImageReader(
	logger *zap.Logger,
	fetchReader buffetch.MessageReader,
) *imageReader {
	return &imageReader{
		logger:      logger.Named(loggerName),
		fetchReader: fetchReader,
		tracer:      otel.GetTracerProvider().Tracer(tracerName),
	}
}

func (i *imageReader) GetImage(
	ctx context.Context,
	container app.EnvStdinContainer,
	messageRef buffetch.MessageRef,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ bufimage.Image, retErr error) {
	ctx, span := i.tracer.Start(ctx, "get_image")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	readCloser, err := i.fetchReader.GetMessageFile(ctx, container, messageRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	data, err := io.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}
	protoImage := &imagev1.Image{}
	var imageFromProtoOptions []bufimage.NewImageForProtoOption
	switch messageEncoding := messageRef.MessageEncoding(); messageEncoding {
	// we have to double parse due to custom options
	// See https://github.com/golang/protobuf/issues/1123
	// TODO: revisit
	case buffetch.MessageEncodingBinpb:
		_, span := i.tracer.Start(ctx, "wire_unmarshal")
		if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, protoImage); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return nil, fmt.Errorf("could not unmarshal image: %v", err)
		}
		span.End()
	case buffetch.MessageEncodingJSON:
		resolver, err := i.bootstrapResolver(ctx, protoencoding.NewJSONUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		_, jsonUnmarshalSpan := i.tracer.Start(ctx, "json_unmarshal")
		if err := protoencoding.NewJSONUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			jsonUnmarshalSpan.RecordError(err)
			jsonUnmarshalSpan.SetStatus(codes.Error, err.Error())
			jsonUnmarshalSpan.End()
			return nil, fmt.Errorf("could not unmarshal image: %v", err)
		}
		jsonUnmarshalSpan.End()
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	case buffetch.MessageEncodingTxtpb:
		resolver, err := i.bootstrapResolver(ctx, protoencoding.NewTxtpbUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		_, txtpbUnmarshalSpan := i.tracer.Start(ctx, "txtpb_unmarshal")
		if err := protoencoding.NewTxtpbUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			txtpbUnmarshalSpan.RecordError(err)
			txtpbUnmarshalSpan.SetStatus(codes.Error, err.Error())
			txtpbUnmarshalSpan.End()
			return nil, fmt.Errorf("could not unmarshal image: %v", err)
		}
		txtpbUnmarshalSpan.End()
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	case buffetch.MessageEncodingYAML:
		resolver, err := i.bootstrapResolver(ctx, protoencoding.NewYAMLUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		_, yamlUnmarshalSpan := i.tracer.Start(ctx, "yaml_unmarshal")
		if err := protoencoding.NewYAMLUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			yamlUnmarshalSpan.RecordError(err)
			yamlUnmarshalSpan.SetStatus(codes.Error, err.Error())
			yamlUnmarshalSpan.End()
			return nil, fmt.Errorf("could not unmarshal image: %v", err)
		}
		yamlUnmarshalSpan.End()
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	default:
		return nil, fmt.Errorf("unknown message encoding: %v", messageEncoding)
	}
	if excludeSourceCodeInfo {
		for _, fileDescriptorProto := range protoImage.File {
			fileDescriptorProto.SourceCodeInfo = nil
		}
	}
	image, err := bufimage.NewImageForProto(protoImage, imageFromProtoOptions...)
	if err != nil {
		return nil, err
	}
	if len(externalDirOrFilePaths) == 0 && len(externalExcludeDirOrFilePaths) == 0 {
		return image, nil
	}
	imagePaths := make([]string, len(externalDirOrFilePaths))
	for i, externalDirOrFilePath := range externalDirOrFilePaths {
		imagePath, err := messageRef.PathForExternalPath(externalDirOrFilePath)
		if err != nil {
			return nil, err
		}
		imagePaths[i] = imagePath
	}
	excludePaths := make([]string, len(externalExcludeDirOrFilePaths))
	for i, excludeDirOrFilePath := range externalExcludeDirOrFilePaths {
		excludePath, err := messageRef.PathForExternalPath(excludeDirOrFilePath)
		if err != nil {
			return nil, err
		}
		excludePaths[i] = excludePath
	}
	if externalDirOrFilePathsAllowNotExist {
		// externalDirOrFilePaths have to be targetPaths
		return bufimage.ImageWithOnlyPathsAllowNotExist(image, imagePaths, excludePaths)
	}
	return bufimage.ImageWithOnlyPaths(image, imagePaths, excludePaths)
}

func (i *imageReader) bootstrapResolver(
	ctx context.Context,
	unresolving protoencoding.Unmarshaler,
	data []byte,
) (protoencoding.Resolver, error) {
	firstProtoImage := &imagev1.Image{}
	_, span := i.tracer.Start(ctx, "bootstrap_unmarshal")
	if err := unresolving.Unmarshal(data, firstProtoImage); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		return nil, fmt.Errorf("could not unmarshal image: %v", err)
	}
	span.End()
	_, newResolverSpan := i.tracer.Start(ctx, "new_resolver")
	resolver, err := protoencoding.NewResolver(firstProtoImage.File...)
	if err != nil {
		newResolverSpan.RecordError(err)
		newResolverSpan.SetStatus(codes.Error, err.Error())
		newResolverSpan.End()
		return nil, err
	}
	newResolverSpan.End()
	return resolver, nil
}
