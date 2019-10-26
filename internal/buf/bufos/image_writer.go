package bufos

import (
	"compress/gzip"
	"context"
	"io"

	"github.com/bufbuild/buf/internal/buf/bufos/internal"
	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/osutil"
	"github.com/bufbuild/buf/internal/pkg/protodescpb"
	"go.uber.org/zap"
)

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
	image bufpb.Image,
) (retErr error) {
	devNull, err := osutil.DevNull()
	if err != nil {
		return err
	}
	// stop short if we have /dev/null equivalent for performance
	if value == devNull {
		return nil
	}
	inputRef, err := i.inputRefParser.ParseInputRef(value, false, true)
	if err != nil {
		return err
	}
	i.logger.Debug("parse", zap.Any("input_ref", inputRef), zap.Stringer("format", inputRef.Format))
	// we now know the format this is only one of FormatBin, FormatBinGz, FormatJSON, FormatJSONGz

	var marshaler protodescpb.Marshaler = image
	if asFileDescriptorSet {
		marshaler, err = image.ToFileDescriptorSet()
		if err != nil {
			return err
		}
	}

	var data []byte
	switch inputRef.Format {
	case internal.FormatJSON, internal.FormatJSONGz:
		data, err = marshaler.MarshalJSON()
		if err != nil {
			return err
		}
	default:
		data, err = marshaler.MarshalWire()
		if err != nil {
			return err
		}
	}

	writeCloser, err := osutil.WriteCloserForFilePath(stdout, inputRef.Path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errs.Append(retErr, writeCloser.Close())
	}()

	switch inputRef.Format {
	case internal.FormatBinGz, internal.FormatJSONGz:
		gzipWriteCloser := gzip.NewWriter(writeCloser)
		defer func() {
			retErr = errs.Append(retErr, gzipWriteCloser.Close())
		}()
		_, err = gzipWriteCloser.Write(data)
		return err
	default:
		_, err = writeCloser.Write(data)
		return err
	}
}
