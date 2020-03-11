package bufos

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"

	"github.com/bufbuild/buf/internal/buf/bufos/internal"
	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/cli/clios"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var jsonMarshaler = &jsonpb.Marshaler{}

type imageWriter struct {
	logger         *zap.Logger
	inputRefParser internal.InputRefParser
}

func newImageWriter(
	logger *zap.Logger,
	valueFlagName string,
) *imageWriter {
	return &imageWriter{
		logger: logger.Named("bufos"),
		inputRefParser: internal.NewInputRefParser(
			valueFlagName,
		),
	}
}

func (i *imageWriter) WriteImage(
	ctx context.Context,
	stdout io.Writer,
	value string,
	asFileDescriptorSet bool,
	image *imagev1beta1.Image,
) (retErr error) {
	if err := extimage.ValidateImage(image); err != nil {
		return err
	}
	// stop short if we have /dev/null equivalent for performance
	if value == clios.DevNull {
		return nil
	}
	inputRef, err := i.inputRefParser.ParseInputRef(value, false, true)
	if err != nil {
		return err
	}
	i.logger.Debug("parse", zap.Any("input_ref", inputRef), zap.Stringer("format", inputRef.Format))
	// we now know the format this is only one of FormatBin, FormatBinGz, FormatJSON, FormatJSONGz

	var message proto.Message = image
	if asFileDescriptorSet {
		message, err = extimage.ImageToFileDescriptorSet(image)
		if err != nil {
			return err
		}
	}

	var data []byte
	switch inputRef.Format {
	case internal.FormatJSON, internal.FormatJSONGz:
		data, err = marshalJSON(message)
		if err != nil {
			return err
		}
	default:
		data, err = proto.Marshal(message)
		if err != nil {
			return err
		}
	}

	writeCloser, err := clios.WriteCloserForFilePath(stdout, inputRef.Path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeCloser.Close())
	}()

	switch inputRef.Format {
	case internal.FormatBinGz, internal.FormatJSONGz:
		gzipWriteCloser := gzip.NewWriter(writeCloser)
		defer func() {
			retErr = multierr.Append(retErr, gzipWriteCloser.Close())
		}()
		_, err = gzipWriteCloser.Write(data)
		return err
	default:
		_, err = writeCloser.Write(data)
		return err
	}
}

func marshalJSON(message proto.Message) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonMarshaler.Marshal(buffer, message); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
