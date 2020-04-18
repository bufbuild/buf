package bufos

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	"github.com/bufbuild/buf/internal/buf/ext/extio"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	iov1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/io/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/util/utilproto"
	"github.com/golang/protobuf/proto"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type imageWriter struct {
	logger        *zap.Logger
	valueFlagName string
}

func newImageWriter(
	logger *zap.Logger,
	valueFlagName string,
) *imageWriter {
	return &imageWriter{
		logger:        logger.Named("bufos"),
		valueFlagName: valueFlagName,
	}
}

func (i *imageWriter) WriteImage(
	ctx context.Context,
	stdoutContainer app.StdoutContainer,
	value string,
	asFileDescriptorSet bool,
	image *imagev1beta1.Image,
) (retErr error) {
	if err := extimage.ValidateImage(image); err != nil {
		return err
	}
	imageRef, err := extio.ParseImageRef(value)
	if err != nil {
		return fmt.Errorf("%s: %v", i.valueFlagName, err)
	}
	i.logger.Debug("write", zap.Any("image_ref", imageRef))

	var message proto.Message = image
	if asFileDescriptorSet {
		message, err = extimage.ImageToFileDescriptorSet(image)
		if err != nil {
			return err
		}
	}
	var data []byte
	switch imageRef.ImageFormat {
	case iov1beta1.ImageFormat_IMAGE_FORMAT_BIN, iov1beta1.ImageFormat_IMAGE_FORMAT_BINGZ:
		data, err = utilproto.MarshalWireDeterministic(message)
		if err != nil {
			return err
		}
	case iov1beta1.ImageFormat_IMAGE_FORMAT_JSON, iov1beta1.ImageFormat_IMAGE_FORMAT_JSONGZ:
		data, err = utilproto.MarshalJSON(message)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown image format: %v", imageRef.ImageFormat)
	}

	var writer io.Writer
	switch imageRef.FileScheme {
	case iov1beta1.FileScheme_FILE_SCHEME_HTTP:
		return fmt.Errorf("%s: cannot write to http", i.valueFlagName)
	case iov1beta1.FileScheme_FILE_SCHEME_HTTPS:
		return fmt.Errorf("%s: cannot write to https", i.valueFlagName)
	case iov1beta1.FileScheme_FILE_SCHEME_NULL:
		// stop short if we have /dev/null equivalent for performance
		return nil
	case iov1beta1.FileScheme_FILE_SCHEME_STDIO:
		writer = stdoutContainer.Stdout()
	case iov1beta1.FileScheme_FILE_SCHEME_FILE:
		file, err := os.Create(imageRef.Path)
		if err != nil {
			return err
		}
		defer func() {
			retErr = multierr.Append(retErr, file.Close())
		}()
		writer = file
	default:
		return fmt.Errorf("unknown file scheme: %v", imageRef.FileScheme)
	}

	switch imageRef.ImageFormat {
	case iov1beta1.ImageFormat_IMAGE_FORMAT_BINGZ, iov1beta1.ImageFormat_IMAGE_FORMAT_JSONGZ:
		gzipWriteCloser := gzip.NewWriter(writer)
		defer func() {
			retErr = multierr.Append(retErr, gzipWriteCloser.Close())
		}()
		_, err = gzipWriteCloser.Write(data)
		return err
	default:
		_, err = writer.Write(data)
		return err
	}
}
