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

package buffetch

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/fetch"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"go.uber.org/zap"
)

type refParser struct {
	logger         *zap.Logger
	fetchRefParser fetch.RefParser
}

func newRefParser(
	logger *zap.Logger,
) *refParser {
	return &refParser{
		logger: logger,
		fetchRefParser: fetch.NewRefParser(
			logger,
			fetch.WithRawRefProcessor(processRawRef),
			fetch.WithSingleFormat(formatBin),
			fetch.WithSingleFormat(formatJSON),
			fetch.WithSingleFormat(
				formatBingz,
				fetch.WithSingleDefaultCompressionType(
					fetch.CompressionTypeGzip,
				),
			),
			fetch.WithSingleFormat(
				formatJSONGZ,
				fetch.WithSingleDefaultCompressionType(
					fetch.CompressionTypeGzip,
				),
			),
			fetch.WithArchiveFormat(
				formatTar,
				fetch.ArchiveTypeTar,
			),
			fetch.WithArchiveFormat(
				formatTargz,
				fetch.ArchiveTypeTar,
				fetch.WithArchiveDefaultCompressionType(
					fetch.CompressionTypeGzip,
				),
			),
			fetch.WithGitFormat(formatGit),
			fetch.WithDirFormat(formatDir),
		),
	}
}

func (a *refParser) GetRef(
	ctx context.Context,
	value string,
) (Ref, error) {
	defer instrument.Start(a.logger, "get_ref").End()
	parsedRef, err := a.getParsedRef(ctx, value, allFormats)
	if err != nil {
		return nil, err
	}
	switch t := parsedRef.(type) {
	case fetch.ParsedSingleRef:
		imageEncoding, err := parseImageEncoding(t.Format())
		if err != nil {
			return nil, err
		}
		return newImageRef(t, imageEncoding), nil
	case fetch.ParsedArchiveRef:
		return newSourceRef(t), nil
	case fetch.ParsedDirRef:
		return newSourceRef(t), nil
	case fetch.ParsedGitRef:
		return newSourceRef(t), nil
	default:
		return nil, fmt.Errorf("known ParsedRef type: %T", parsedRef)
	}
}

func (a *refParser) GetImageRef(
	ctx context.Context,
	value string,
) (ImageRef, error) {
	defer instrument.Start(a.logger, "get_image_ref").End()
	parsedRef, err := a.getParsedRef(ctx, value, imageFormats)
	if err != nil {
		return nil, err
	}
	parsedSingleRef, ok := parsedRef.(fetch.ParsedSingleRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid ParsedRef type for image: %T", parsedRef)
	}
	imageEncoding, err := parseImageEncoding(parsedSingleRef.Format())
	if err != nil {
		return nil, err
	}
	return newImageRef(parsedSingleRef, imageEncoding), nil
}

func (a *refParser) GetSourceRef(
	ctx context.Context,
	value string,
) (SourceRef, error) {
	defer instrument.Start(a.logger, "get_source_ref").End()
	parsedRef, err := a.getParsedRef(ctx, value, sourceFormats)
	if err != nil {
		return nil, err
	}
	parsedBucketRef, ok := parsedRef.(fetch.ParsedBucketRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid ParsedRef type for source: %T", parsedRef)
	}
	return newSourceRef(parsedBucketRef), nil
}

func (a *refParser) getParsedRef(
	ctx context.Context,
	value string,
	allowedFormats []string,
) (fetch.ParsedRef, error) {
	parsedRef, err := a.fetchRefParser.GetParsedRef(
		ctx,
		value,
		fetch.WithAllowedFormats(allowedFormats...),
	)
	if err != nil {
		return nil, err
	}
	a.checkDeprecated(parsedRef)
	return parsedRef, nil
}

func (a *refParser) checkDeprecated(parsedRef fetch.ParsedRef) {
	format := parsedRef.Format()
	if replacementFormat, ok := deprecatedCompressionFormatToReplacementFormat[format]; ok {
		a.logger.Sugar().Warnf(
			`Format %q is deprecated. Use "format=%s,compression=gz" instead. This will continue to work forever, but updating is recommended.`,
			format,
			replacementFormat,
		)
	}
}

func processRawRef(rawRef *fetch.RawRef) error {
	// if format option is not set and path is "-", default to bin
	var format string
	var compressionType fetch.CompressionType
	if rawRef.Path == "-" || rawRef.Path == app.DevNullFilePath {
		format = formatBin
	} else {
		switch filepath.Ext(rawRef.Path) {
		case ".bin":
			format = formatBin
		case ".json":
			format = formatJSON
		case ".tar":
			format = formatTar
		case ".gz":
			compressionType = fetch.CompressionTypeGzip
			switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
			case ".bin":
				format = formatBin
			case ".json":
				format = formatJSON
			case ".tar":
				format = formatTar
			default:
				return fmt.Errorf("path %q had .gz extension with unknown format", rawRef.Path)
			}
		case ".tgz":
			format = formatTar
			compressionType = fetch.CompressionTypeGzip
		case ".git":
			format = formatGit
		default:
			format = formatDir
		}
	}
	rawRef.Format = format
	rawRef.CompressionType = compressionType
	return nil
}

func parseImageEncoding(format string) (ImageEncoding, error) {
	switch format {
	case formatBin, formatBingz:
		return ImageEncodingBin, nil
	case formatJSON, formatJSONGZ:
		return ImageEncodingJSON, nil
	default:
		return 0, fmt.Errorf("invalid format for image: %q", format)
	}
}
