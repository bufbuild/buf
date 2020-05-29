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
		logger: logger.Named("buffetch"),
		fetchRefParser: fetch.NewRefParser(
			logger,
			fetch.WithFormatParser(parseFormat),
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
	parsedRef, err := a.fetchRefParser.GetParsedRef(
		ctx,
		value,
		fetch.WithAllowedFormats(allFormats...),
	)
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
	parsedRef, err := a.fetchRefParser.GetParsedRef(
		ctx,
		value,
		fetch.WithAllowedFormats(imageFormats...),
	)
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
	parsedRef, err := a.fetchRefParser.GetParsedRef(
		ctx,
		value,
		fetch.WithAllowedFormats(sourceFormats...),
	)
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

func parseFormat(rawPath string) (string, error) {
	// if format option is not set and path is "-", default to bin
	if rawPath == "-" || rawPath == app.DevNullFilePath {
		return formatBin, nil
	}
	switch filepath.Ext(rawPath) {
	case ".bin":
		return formatBin, nil
	case ".json":
		return formatJSON, nil
	case ".tar":
		return formatTar, nil
	case ".gz":
		switch filepath.Ext(strings.TrimSuffix(rawPath, filepath.Ext(rawPath))) {
		case ".bin":
			return formatBingz, nil
		case ".json":
			return formatJSONGZ, nil
		case ".tar":
			return formatTargz, nil
		default:
			return "", fmt.Errorf("path %q had .gz extension with unknown format", rawPath)
		}
	case ".tgz":
		return formatTargz, nil
	case ".git":
		return formatGit, nil
	default:
		return formatDir, nil
	}
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
