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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/fetch"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"go.uber.org/zap"
)

type refParser struct {
	logger         *zap.Logger
	fetchRefParser fetch.RefParser

	workdir     string
	workdirErr  error
	workdirOnce sync.Once
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
	fetchRef, err := a.fetchRefParser.GetRef(
		ctx,
		value,
		fetch.WithAllowedFormats(allFormats...),
	)
	if err != nil {
		return nil, err
	}
	switch t := fetchRef.(type) {
	case fetch.SingleRef:
		imageEncoding, err := parseImageEncoding(t.Format())
		if err != nil {
			return nil, err
		}
		return newImageRef(t, imageEncoding), nil
	case fetch.ArchiveRef:
		workdir, err := a.getWorkdir()
		if err != nil {
			return nil, err
		}
		return newSourceRef(t, workdir), nil
	case fetch.DirRef:
		workdir, err := a.getWorkdir()
		if err != nil {
			return nil, err
		}
		return newSourceRef(t, workdir), nil
	case fetch.GitRef:
		workdir, err := a.getWorkdir()
		if err != nil {
			return nil, err
		}
		return newSourceRef(t, workdir), nil
	default:
		return nil, fmt.Errorf("known Ref type: %T", fetchRef)
	}
}

func (a *refParser) GetImageRef(
	ctx context.Context,
	value string,
) (ImageRef, error) {
	defer instrument.Start(a.logger, "get_image_ref").End()
	ref, err := a.fetchRefParser.GetRef(
		ctx,
		value,
		fetch.WithAllowedFormats(imageFormats...),
	)
	if err != nil {
		return nil, err
	}
	singleRef, ok := ref.(fetch.SingleRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid Ref type for image: %T", ref)
	}
	imageEncoding, err := parseImageEncoding(singleRef.Format())
	if err != nil {
		return nil, err
	}
	return newImageRef(singleRef, imageEncoding), nil
}

func (a *refParser) GetSourceRef(
	ctx context.Context,
	value string,
) (SourceRef, error) {
	defer instrument.Start(a.logger, "get_source_ref").End()
	ref, err := a.fetchRefParser.GetRef(
		ctx,
		value,
		fetch.WithAllowedFormats(sourceFormats...),
	)
	if err != nil {
		return nil, err
	}
	bucketRef, ok := ref.(fetch.BucketRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid Ref type for source: %T", ref)
	}
	workdir, err := a.getWorkdir()
	if err != nil {
		return nil, err
	}
	return newSourceRef(bucketRef, workdir), nil
}

func (a *refParser) getWorkdir() (string, error) {
	a.workdirOnce.Do(a.populateWorkdir)
	if a.workdirErr != nil {
		return "", a.workdirErr
	}
	if a.workdir == "" {
		return "", errors.New("could not determine working directory")
	}
	return a.workdir, nil
}

func (a *refParser) populateWorkdir() {
	a.workdir, a.workdirErr = os.Getwd()
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
