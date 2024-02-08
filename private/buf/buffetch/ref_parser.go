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

package buffetch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

type refParser struct {
	logger         *zap.Logger
	fetchRefParser internal.RefParser
}

func newRefParser(logger *zap.Logger) *refParser {
	return &refParser{
		logger: logger,
		fetchRefParser: internal.NewRefParser(
			logger,
			internal.WithRawRefProcessor(processRawRef),
			internal.WithSingleFormat(formatBin),
			internal.WithSingleFormat(formatBinpb),
			internal.WithSingleFormat(
				formatJSON,
				internal.WithSingleCustomOptionKey(useProtoNamesKey),
				internal.WithSingleCustomOptionKey(useEnumNumbersKey),
			),
			internal.WithSingleFormat(formatTxtpb),
			internal.WithSingleFormat(
				formatYAML,
				internal.WithSingleCustomOptionKey(useProtoNamesKey),
				internal.WithSingleCustomOptionKey(useEnumNumbersKey),
			),
			internal.WithSingleFormat(
				formatBingz,
				internal.WithSingleDefaultCompressionType(
					internal.CompressionTypeGzip,
				),
			),
			internal.WithSingleFormat(
				formatJSONGZ,
				internal.WithSingleDefaultCompressionType(
					internal.CompressionTypeGzip,
				),
			),
			internal.WithArchiveFormat(
				formatTar,
				internal.ArchiveTypeTar,
			),
			internal.WithArchiveFormat(
				formatTargz,
				internal.ArchiveTypeTar,
				internal.WithArchiveDefaultCompressionType(
					internal.CompressionTypeGzip,
				),
			),
			internal.WithArchiveFormat(
				formatZip,
				internal.ArchiveTypeZip,
			),
			internal.WithGitFormat(formatGit),
			internal.WithDirFormat(formatDir),
			internal.WithModuleFormat(formatMod),
			internal.WithProtoFileFormat(formatProtoFile),
		),
	}
}

func newMessageRefParser(logger *zap.Logger, options ...MessageRefParserOption) *refParser {
	messageRefParserOptions := newMessageRefParserOptions()
	for _, option := range options {
		option(messageRefParserOptions)
	}
	return &refParser{
		logger: logger,
		fetchRefParser: internal.NewRefParser(
			logger,
			internal.WithRawRefProcessor(newProcessRawRefMessage(messageRefParserOptions.defaultMessageEncoding)),
			internal.WithSingleFormat(formatBin),
			internal.WithSingleFormat(formatBinpb),
			internal.WithSingleFormat(
				formatJSON,
				internal.WithSingleCustomOptionKey(useProtoNamesKey),
				internal.WithSingleCustomOptionKey(useEnumNumbersKey),
			),
			internal.WithSingleFormat(formatTxtpb),
			internal.WithSingleFormat(
				formatYAML,
				internal.WithSingleCustomOptionKey(useProtoNamesKey),
				internal.WithSingleCustomOptionKey(useEnumNumbersKey),
			),
			internal.WithSingleFormat(
				formatBingz,
				internal.WithSingleDefaultCompressionType(
					internal.CompressionTypeGzip,
				),
			),
			internal.WithSingleFormat(
				formatJSONGZ,
				internal.WithSingleDefaultCompressionType(
					internal.CompressionTypeGzip,
				),
			),
		),
	}
}

func newSourceRefParser(logger *zap.Logger) *refParser {
	return &refParser{
		logger: logger,
		fetchRefParser: internal.NewRefParser(
			logger,
			internal.WithRawRefProcessor(processRawRefSource),
			internal.WithArchiveFormat(
				formatTar,
				internal.ArchiveTypeTar,
			),
			internal.WithArchiveFormat(
				formatTargz,
				internal.ArchiveTypeTar,
				internal.WithArchiveDefaultCompressionType(
					internal.CompressionTypeGzip,
				),
			),
			internal.WithArchiveFormat(
				formatZip,
				internal.ArchiveTypeZip,
			),
			internal.WithGitFormat(formatGit),
			internal.WithDirFormat(formatDir),
			internal.WithProtoFileFormat(formatProtoFile),
		),
	}
}

func newDirRefParser(logger *zap.Logger) *refParser {
	return &refParser{
		logger: logger,
		fetchRefParser: internal.NewRefParser(
			logger,
			internal.WithRawRefProcessor(processRawRefDir),
			internal.WithDirFormat(formatDir),
		),
	}
}

func newDirOrProtoFileRefParser(logger *zap.Logger) *refParser {
	return &refParser{
		logger: logger,
		fetchRefParser: internal.NewRefParser(
			logger,
			internal.WithRawRefProcessor(processRawRefDirOrProtoFile),
			internal.WithDirFormat(formatDir),
			internal.WithProtoFileFormat(formatProtoFile),
		),
	}
}

func newModuleRefParser(logger *zap.Logger) *refParser {
	return &refParser{
		logger: logger,
		fetchRefParser: internal.NewRefParser(
			logger,
			internal.WithRawRefProcessor(processRawRefModule),
			internal.WithModuleFormat(formatMod),
		),
	}
}

func newSourceOrModuleRefParser(logger *zap.Logger) *refParser {
	return &refParser{
		logger: logger,
		fetchRefParser: internal.NewRefParser(
			logger,
			internal.WithRawRefProcessor(processRawRefSourceOrModule),
			internal.WithArchiveFormat(
				formatTar,
				internal.ArchiveTypeTar,
			),
			internal.WithArchiveFormat(
				formatTargz,
				internal.ArchiveTypeTar,
				internal.WithArchiveDefaultCompressionType(
					internal.CompressionTypeGzip,
				),
			),
			internal.WithArchiveFormat(
				formatZip,
				internal.ArchiveTypeZip,
			),
			internal.WithGitFormat(formatGit),
			internal.WithDirFormat(formatDir),
			internal.WithModuleFormat(formatMod),
			internal.WithProtoFileFormat(formatProtoFile),
		),
	}
}

func (a *refParser) GetRef(
	ctx context.Context,
	value string,
) (Ref, error) {
	parsedRef, err := a.getParsedRef(ctx, value, allFormats)
	if err != nil {
		return nil, err
	}
	switch t := parsedRef.(type) {
	case internal.ParsedSingleRef:
		messageEncoding, err := parseMessageEncoding(t.Format())
		if err != nil {
			return nil, err
		}
		return newMessageRef(t, messageEncoding)
	case internal.ParsedArchiveRef:
		return newSourceRef(t), nil
	case internal.ParsedDirRef:
		return newDirRef(t), nil
	case internal.ParsedGitRef:
		return newSourceRef(t), nil
	case internal.ParsedModuleRef:
		return newModuleRef(t), nil
	case internal.ProtoFileRef:
		return newProtoFileRef(t), nil
	default:
		return nil, fmt.Errorf("unknown ParsedRef type: %T", parsedRef)
	}
}

func (a *refParser) GetRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
) (Ref, error) {
	parsedRef, err := a.getParsedRefForInputConfig(ctx, inputConfig, allFormats)
	if err != nil {
		return nil, err
	}
	switch t := parsedRef.(type) {
	case internal.ParsedSingleRef:
		messageEncoding, err := parseMessageEncoding(t.Format())
		if err != nil {
			return nil, err
		}
		return newMessageRef(t, messageEncoding)
	case internal.ParsedArchiveRef:
		return newSourceRef(t), nil
	case internal.ParsedDirRef:
		return newDirRef(t), nil
	case internal.ParsedGitRef:
		return newSourceRef(t), nil
	case internal.ParsedModuleRef:
		return newModuleRef(t), nil
	case internal.ProtoFileRef:
		return newProtoFileRef(t), nil
	default:
		return nil, fmt.Errorf("unknown ParsedRef type: %T", parsedRef)
	}
}

func (a *refParser) GetSourceOrModuleRef(
	ctx context.Context,
	value string,
) (SourceOrModuleRef, error) {
	parsedRef, err := a.getParsedRef(ctx, value, sourceOrModuleFormats)
	if err != nil {
		return nil, err
	}
	switch t := parsedRef.(type) {
	case internal.ParsedSingleRef:
		return nil, fmt.Errorf("invalid ParsedRef type for source or module: %T", parsedRef)
	case internal.ParsedArchiveRef:
		return newSourceRef(t), nil
	case internal.ParsedDirRef:
		return newDirRef(t), nil
	case internal.ParsedGitRef:
		return newSourceRef(t), nil
	case internal.ParsedModuleRef:
		return newModuleRef(t), nil
	case internal.ProtoFileRef:
		return newProtoFileRef(t), nil
	default:
		return nil, fmt.Errorf("unknown ParsedRef type: %T", parsedRef)
	}
}

func (a *refParser) GetSourceOrModuleRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
) (SourceOrModuleRef, error) {
	parsedRef, err := a.getParsedRefForInputConfig(ctx, inputConfig, sourceOrModuleFormats)
	if err != nil {
		return nil, err
	}
	switch t := parsedRef.(type) {
	case internal.ParsedSingleRef:
		return nil, fmt.Errorf("invalid ParsedRef type for source or module: %T", parsedRef)
	case internal.ParsedArchiveRef:
		return newSourceRef(t), nil
	case internal.ParsedDirRef:
		return newDirRef(t), nil
	case internal.ParsedGitRef:
		return newSourceRef(t), nil
	case internal.ParsedModuleRef:
		return newModuleRef(t), nil
	case internal.ProtoFileRef:
		return newProtoFileRef(t), nil
	default:
		return nil, fmt.Errorf("unknown ParsedRef type: %T", parsedRef)
	}
}

func (a *refParser) GetMessageRef(
	ctx context.Context,
	value string,
) (MessageRef, error) {
	parsedRef, err := a.getParsedRef(ctx, value, messageFormats)
	if err != nil {
		return nil, err
	}
	parsedSingleRef, ok := parsedRef.(internal.ParsedSingleRef)
	if !ok {
		return nil, fmt.Errorf("invalid ParsedRef type for message: %T", parsedRef)
	}
	messageEncoding, err := parseMessageEncoding(parsedSingleRef.Format())
	if err != nil {
		return nil, err
	}
	return newMessageRef(parsedSingleRef, messageEncoding)
}

func (a *refParser) GetMessageRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
) (MessageRef, error) {
	parsedRef, err := a.getParsedRefForInputConfig(ctx, inputConfig, messageFormats)
	if err != nil {
		return nil, err
	}
	parsedSingleRef, ok := parsedRef.(internal.ParsedSingleRef)
	if !ok {
		return nil, fmt.Errorf("invalid ParsedRef type for message: %T", parsedRef)
	}
	messageEncoding, err := parseMessageEncoding(parsedSingleRef.Format())
	if err != nil {
		return nil, err
	}
	return newMessageRef(parsedSingleRef, messageEncoding)
}

func (a *refParser) GetSourceRef(
	ctx context.Context,
	value string,
) (SourceRef, error) {
	parsedRef, err := a.getParsedRef(ctx, value, sourceFormats)
	if err != nil {
		return nil, err
	}
	parsedBucketRef, ok := parsedRef.(internal.ParsedBucketRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid ParsedRef type for source: %T", parsedRef)
	}
	return newSourceRef(parsedBucketRef), nil
}

func (a *refParser) GetSourceRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
) (SourceRef, error) {
	parsedRef, err := a.getParsedRefForInputConfig(ctx, inputConfig, sourceFormats)
	if err != nil {
		return nil, err
	}
	parsedBucketRef, ok := parsedRef.(internal.ParsedBucketRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid ParsedRef type for source: %T", parsedRef)
	}
	return newSourceRef(parsedBucketRef), nil
}

func (a *refParser) GetDirRef(
	ctx context.Context,
	value string,
) (DirRef, error) {
	parsedRef, err := a.getParsedRef(ctx, value, dirFormats)
	if err != nil {
		return nil, err
	}
	parsedDirRef, ok := parsedRef.(internal.ParsedDirRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid ParsedRef type for source: %T", parsedRef)
	}
	return newDirRef(parsedDirRef), nil
}

func (a *refParser) GetDirRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
) (DirRef, error) {
	parsedRef, err := a.getParsedRefForInputConfig(ctx, inputConfig, dirFormats)
	if err != nil {
		return nil, err
	}
	parsedDirRef, ok := parsedRef.(internal.ParsedDirRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid ParsedRef type for source: %T", parsedRef)
	}
	return newDirRef(parsedDirRef), nil
}

func (a *refParser) GetDirOrProtoFileRef(
	ctx context.Context,
	value string,
) (DirOrProtoFileRef, error) {
	parsedRef, err := a.getParsedRef(ctx, value, dirOrProtoFileFormats)
	if err != nil {
		return nil, err
	}
	switch t := parsedRef.(type) {
	case internal.ParsedDirRef:
		return newDirRef(t), nil
	case internal.ProtoFileRef:
		return newProtoFileRef(t), nil
	default:
		return nil, fmt.Errorf("invalid ParsedRef type: %T", parsedRef)
	}
}

func (a *refParser) GetDirOrProtoFileRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
) (DirOrProtoFileRef, error) {
	parsedRef, err := a.getParsedRefForInputConfig(ctx, inputConfig, dirOrProtoFileFormats)
	if err != nil {
		return nil, err
	}
	switch t := parsedRef.(type) {
	case internal.ParsedDirRef:
		return newDirRef(t), nil
	case internal.ProtoFileRef:
		return newProtoFileRef(t), nil
	default:
		return nil, fmt.Errorf("invalid ParsedRef type: %T", parsedRef)
	}
}

func (a *refParser) GetModuleRef(
	ctx context.Context,
	value string,
) (ModuleRef, error) {
	parsedRef, err := a.getParsedRef(ctx, value, moduleFormats)
	if err != nil {
		return nil, err
	}
	parsedModuleRef, ok := parsedRef.(internal.ParsedModuleRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid ParsedRef type for source: %T", parsedRef)
	}
	return newModuleRef(parsedModuleRef), nil
}

func (a *refParser) GetModuleRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
) (ModuleRef, error) {
	parsedRef, err := a.getParsedRefForInputConfig(ctx, inputConfig, moduleFormats)
	if err != nil {
		return nil, err
	}
	parsedModuleRef, ok := parsedRef.(internal.ParsedModuleRef)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("invalid ParsedRef type for source: %T", parsedRef)
	}
	return newModuleRef(parsedModuleRef), nil
}

// TODO FUTURE: rename to getParsedRefForString
func (a *refParser) getParsedRef(
	ctx context.Context,
	value string,
	allowedFormats []string,
) (internal.ParsedRef, error) {
	parsedRef, err := a.fetchRefParser.GetParsedRef(
		ctx,
		value,
		internal.WithAllowedFormats(allowedFormats...),
	)
	if err != nil {
		return nil, err
	}
	a.checkDeprecated(parsedRef)
	return parsedRef, nil
}

func (a *refParser) getParsedRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
	allowedFormats []string,
) (internal.ParsedRef, error) {
	parsedRef, err := a.fetchRefParser.GetParsedRefForInputConfig(
		ctx,
		inputConfig,
		internal.WithAllowedFormats(allowedFormats...),
	)
	if err != nil {
		return nil, err
	}
	a.checkDeprecated(parsedRef)
	return parsedRef, nil
}

func (a *refParser) checkDeprecated(parsedRef internal.ParsedRef) {
	format := parsedRef.Format()
	if replacementFormat, ok := deprecatedCompressionFormatToReplacementFormat[format]; ok {
		a.logger.Sugar().Warnf(
			`Format %q is deprecated. Use "format=%s,compression=gzip" instead. This will continue to work forever, but updating is recommended.`,
			format,
			replacementFormat,
		)
	}
}

func processRawRef(rawRef *internal.RawRef) error {
	// if format option is not set and path is "-", default to bin
	var format string
	var compressionType internal.CompressionType
	if rawRef.Path == "-" || app.IsDevPath(rawRef.Path) {
		format = formatBinpb
	} else {
		switch filepath.Ext(rawRef.Path) {
		case ".bin", ".binpb":
			format = formatBinpb
		case ".json":
			format = formatJSON
		case ".tar":
			format = formatTar
		case ".txtpb":
			format = formatTxtpb
		case ".yaml":
			format = formatYAML
		case ".zip":
			format = formatZip
		case ".gz":
			compressionType = internal.CompressionTypeGzip
			switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
			case ".bin", ".binpb":
				format = formatBinpb
			case ".json":
				format = formatJSON
			case ".tar":
				format = formatTar
			case ".txtpb":
				format = formatTxtpb
			case ".yaml":
				format = formatYAML
			default:
				return fmt.Errorf("path %q had .gz extension with unknown format", rawRef.Path)
			}
		case ".zst":
			compressionType = internal.CompressionTypeZstd
			switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
			case ".bin", ".binpb":
				format = formatBinpb
			case ".json":
				format = formatJSON
			case ".tar":
				format = formatTar
			case ".txtpb":
				format = formatTxtpb
			case ".yaml":
				format = formatYAML
			default:
				return fmt.Errorf("path %q had .zst extension with unknown format", rawRef.Path)
			}
		case ".tgz":
			format = formatTar
			compressionType = internal.CompressionTypeGzip
		case ".git":
			format = formatGit
		case ".proto":
			fileInfo, err := os.Stat(rawRef.Path)
			if err == nil && fileInfo.IsDir() {
				format = formatDir
			} else {
				format = formatProtoFile
			}
		default:
			var err error
			format, err = assumeModuleOrDir(rawRef.Path)
			if err != nil {
				return err
			}
		}
	}
	rawRef.Format = format
	rawRef.CompressionType = compressionType
	return nil
}

func processRawRefSource(rawRef *internal.RawRef) error {
	var format string
	var compressionType internal.CompressionType
	switch filepath.Ext(rawRef.Path) {
	case ".tar":
		format = formatTar
	case ".zip":
		format = formatZip
	case ".gz":
		compressionType = internal.CompressionTypeGzip
		switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
		case ".tar":
			format = formatTar
		default:
			return fmt.Errorf("path %q had .gz extension with unknown format", rawRef.Path)
		}
	case ".zst":
		compressionType = internal.CompressionTypeZstd
		switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
		case ".tar":
			format = formatTar
		default:
			return fmt.Errorf("path %q had .zst extension with unknown format", rawRef.Path)
		}
	case ".tgz":
		format = formatTar
		compressionType = internal.CompressionTypeGzip
	case ".git":
		format = formatGit
	case ".proto":
		fileInfo, err := os.Stat(rawRef.Path)
		if err == nil && fileInfo.IsDir() {
			format = formatDir
		} else {
			format = formatProtoFile
		}
	default:
		format = formatDir
	}
	rawRef.Format = format
	rawRef.CompressionType = compressionType
	return nil
}

func processRawRefDir(rawRef *internal.RawRef) error {
	rawRef.Format = formatDir
	return nil
}

func processRawRefDirOrProtoFile(rawRef *internal.RawRef) error {
	var format string
	if rawRef.Path == "-" || app.IsDevPath(rawRef.Path) {
		format = formatProtoFile
	} else {
		switch filepath.Ext(rawRef.Path) {
		case ".proto":
			fileInfo, err := os.Stat(rawRef.Path)
			if err == nil && fileInfo.IsDir() {
				format = formatDir
			} else {
				format = formatProtoFile
			}
		default:
			var err error
			format, err = assumeModuleOrDir(rawRef.Path)
			if err != nil {
				return err
			}
			if format == formatMod {
				return ErrModuleFormatDetectedForDirOrProtoFileRef
			}
		}
	}
	rawRef.Format = format
	return nil
}

func processRawRefSourceOrModule(rawRef *internal.RawRef) error {
	var format string
	var compressionType internal.CompressionType
	switch filepath.Ext(rawRef.Path) {
	case ".tar":
		format = formatTar
	case ".zip":
		format = formatZip
	case ".gz":
		compressionType = internal.CompressionTypeGzip
		switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
		case ".tar":
			format = formatTar
		default:
			return fmt.Errorf("path %q had .gz extension with unknown format", rawRef.Path)
		}
	case ".zst":
		compressionType = internal.CompressionTypeZstd
		switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
		case ".tar":
			format = formatTar
		default:
			return fmt.Errorf("path %q had .zst extension with unknown format", rawRef.Path)
		}
	case ".tgz":
		format = formatTar
		compressionType = internal.CompressionTypeGzip
	case ".git":
		format = formatGit
	case ".proto":
		fileInfo, err := os.Stat(rawRef.Path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("path provided is not a valid proto file: %s, %w", rawRef.Path, err)
		}
		if fileInfo != nil && fileInfo.IsDir() {
			return fmt.Errorf("path provided is not a valid proto file: a directory named %s already exists", rawRef.Path)
		}
		format = formatProtoFile
	default:
		var err error
		format, err = assumeModuleOrDir(rawRef.Path)
		if err != nil {
			return err
		}
	}
	rawRef.Format = format
	rawRef.CompressionType = compressionType
	return nil
}

func newProcessRawRefMessage(defaultMessageEncoding MessageEncoding) func(*internal.RawRef) error {
	return func(rawRef *internal.RawRef) error {
		defaultFormat, ok := messageEncodingToFormat[defaultMessageEncoding]
		if !ok {
			// This is a system error.
			return syserror.Newf("unknown MessageEncoding: %v", defaultMessageEncoding)
		}
		// if format option is not set and path is "-", default to bin
		var format string
		var compressionType internal.CompressionType
		if rawRef.Path == "-" || app.IsDevNull(rawRef.Path) || app.IsDevStdin(rawRef.Path) || app.IsDevStdout(rawRef.Path) {
			format = defaultFormat
		} else {
			switch filepath.Ext(rawRef.Path) {
			case ".bin", ".binpb":
				format = formatBinpb
			case ".json":
				format = formatJSON
			case ".txtpb":
				format = formatTxtpb
			case ".yaml":
				format = formatYAML
			case ".gz":
				compressionType = internal.CompressionTypeGzip
				switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
				case ".bin", ".binpb":
					format = formatBinpb
				case ".json":
					format = formatJSON
				case ".txtpb":
					format = formatTxtpb
				case ".yaml":
					format = formatYAML
				default:
					return fmt.Errorf("path %q had .gz extension with unknown format", rawRef.Path)
				}
			case ".zst":
				compressionType = internal.CompressionTypeZstd
				switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
				case ".bin", ".binpb":
					format = formatBinpb
				case ".json":
					format = formatJSON
				case ".txtpb":
					format = formatTxtpb
				case ".yaml":
					format = formatYAML
				default:
					return fmt.Errorf("path %q had .zst extension with unknown format", rawRef.Path)
				}
			default:
				format = defaultFormat
			}
		}
		rawRef.Format = format
		rawRef.CompressionType = compressionType
		return nil
	}
}

func processRawRefModule(rawRef *internal.RawRef) error {
	rawRef.Format = formatMod
	return nil
}

func parseMessageEncoding(format string) (MessageEncoding, error) {
	switch format {
	case formatBin, formatBinpb, formatBingz:
		return MessageEncodingBinpb, nil
	case formatJSON, formatJSONGZ:
		return MessageEncodingJSON, nil
	case formatTxtpb:
		return MessageEncodingTxtpb, nil
	case formatYAML:
		return MessageEncodingYAML, nil
	default:
		return 0, fmt.Errorf("invalid format for message: %q", format)
	}
}

// TODO FUTURE: this is a terrible heuristic, and we shouldn't be using what amounts
// to heuristics here (technically this is a documentable rule, but still)
func assumeModuleOrDir(path string) (string, error) {
	if path == "" {
		return "", errors.New("assumeModuleOrDir: no path given")
	}
	if _, err := bufmodule.ParseModuleRef(path); err == nil {
		// this is possible to be a module, check if it is a directory though
		// OK to use os.Stat instead of os.Lstat here
		fileInfo, err := os.Stat(path)
		if err == nil && fileInfo.IsDir() {
			// if we have a directory, assume this is a directory
			return formatDir, nil
		}
		// not a directory, assume module
		return formatMod, nil
	}
	// cannot be parsed into a module, assume dir for here
	return formatDir, nil
}

type messageRefParserOptions struct {
	defaultMessageEncoding MessageEncoding
}

func newMessageRefParserOptions() *messageRefParserOptions {
	return &messageRefParserOptions{
		defaultMessageEncoding: MessageEncodingBinpb,
	}
}
