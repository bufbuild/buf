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

package bufos

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/proto/protoencoding"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type imageWriter struct {
	logger         *zap.Logger
	fetchRefParser buffetch.RefParser
	fetchWriter    buffetch.Writer
}

func newImageWriter(
	logger *zap.Logger,
	fetchRefParser buffetch.RefParser,
	fetchWriter buffetch.Writer,
) *imageWriter {
	return &imageWriter{
		logger:         logger.Named("bufos"),
		fetchRefParser: fetchRefParser,
		fetchWriter:    fetchWriter,
	}
}

func (i *imageWriter) WriteImage(
	ctx context.Context,
	container app.EnvStdoutContainer,
	value string,
	asFileDescriptorSet bool,
	image *imagev1beta1.Image,
	imageWithImports *imagev1beta1.Image,
) (retErr error) {
	defer instrument.Start(i.logger, "write_image").End()
	if err := extimage.ValidateImage(image); err != nil {
		return err
	}
	imageRef, err := i.fetchRefParser.GetImageRef(ctx, value)
	if err != nil {
		return err
	}
	// stop short for performance
	if imageRef.IsNull() {
		return nil
	}
	var message proto.Message = image
	if asFileDescriptorSet {
		message, err = extimage.ImageToFileDescriptorSet(image)
		if err != nil {
			return err
		}
	}
	data, err := i.imageMarshal(message, imageWithImports, imageRef.ImageEncoding())
	if err != nil {
		return err
	}
	writeCloser, err := i.fetchWriter.PutImage(ctx, container, imageRef)
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
	message proto.Message,
	imageWithImports *imagev1beta1.Image,
	imageEncoding buffetch.ImageEncoding,
) ([]byte, error) {
	defer instrument.Start(i.logger, "image_marshal").End()
	switch imageEncoding {
	case buffetch.ImageEncodingBin:
		return protoencoding.NewWireMarshaler().Marshal(message)
	case buffetch.ImageEncodingJSON:
		if imageWithImports == nil {
			return nil, errors.New("cannot serialize image to json without imports present")
		}
		resolver, err := protoencoding.NewResolver(imageWithImports.File...)
		if err != nil {
			return nil, err
		}
		return protoencoding.NewJSONMarshaler(resolver).Marshal(message)
	default:
		return nil, fmt.Errorf("unknown image encoding: %v", imageEncoding)
	}
}
