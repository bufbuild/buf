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

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type imageWriter struct {
	logger      *zap.Logger
	fetchWriter buffetch.Writer
}

func newImageWriter(
	logger *zap.Logger,
	fetchWriter buffetch.Writer,
) *imageWriter {
	return &imageWriter{
		logger:      logger,
		fetchWriter: fetchWriter,
	}
}

func (i *imageWriter) PutImage(
	ctx context.Context,
	container app.EnvStdoutContainer,
	messageRef buffetch.MessageRef,
	image bufimage.Image,
	asFileDescriptorSet bool,
	excludeImports bool,
) (retErr error) {
	ctx, span := otel.GetTracerProvider().Tracer("bufbuild/buf").Start(ctx, "put_image")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	// stop short for performance
	if messageRef.IsNull() {
		return nil
	}
	writeImage := image
	if excludeImports {
		writeImage = bufimage.ImageWithoutImports(image)
	}
	var message proto.Message
	if asFileDescriptorSet {
		message = bufimage.ImageToFileDescriptorSet(writeImage)
	} else {
		message = bufimage.ImageToProtoImage(writeImage)
	}
	data, err := i.imageMarshal(ctx, message, image, messageRef)
	if err != nil {
		return err
	}
	writeCloser, err := i.fetchWriter.PutMessageFile(ctx, container, messageRef)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeCloser.Close())
	}()
	_, err = writeCloser.Write(data)
	return err
}

func (i *imageWriter) imageMarshal(
	ctx context.Context,
	message proto.Message,
	image bufimage.Image,
	messageRef buffetch.MessageRef,
) (_ []byte, retErr error) {
	_, span := otel.GetTracerProvider().Tracer("bufbuild/buf").Start(ctx, "image_marshal")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	switch messageEncoding := messageRef.MessageEncoding(); messageEncoding {
	case buffetch.MessageEncodingBinpb:
		return protoencoding.NewWireMarshaler().Marshal(message)
	case buffetch.MessageEncodingJSON:
		// TODO: verify that image is complete
		resolver, err := protoencoding.NewResolver(
			bufimage.ImageToFileDescriptorProtos(image)...,
		)
		if err != nil {
			return nil, err
		}
		return newJSONMarshaler(resolver, messageRef).Marshal(message)
	case buffetch.MessageEncodingTxtpb:
		// TODO: verify that image is complete
		resolver, err := protoencoding.NewResolver(
			bufimage.ImageToFileDescriptorProtos(image)...,
		)
		if err != nil {
			return nil, err
		}
		return protoencoding.NewTxtpbMarshaler(resolver).Marshal(message)
	case buffetch.MessageEncodingYAML:
		resolver, err := protoencoding.NewResolver(
			bufimage.ImageToFileDescriptorProtos(
				image,
			)...,
		)
		if err != nil {
			return nil, err
		}
		return newYAMLMarshaler(resolver, messageRef).Marshal(message)
	default:
		return nil, fmt.Errorf("unknown message encoding: %v", messageEncoding)
	}
}
