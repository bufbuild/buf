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
	"io"
	"net/http"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/zap"
)

const (
	// MessageEncodingBinpb is the binary message encoding.
	MessageEncodingBinpb MessageEncoding = iota + 1
	// MessageEncodingJSON is the JSON message encoding.
	MessageEncodingJSON
	// MessageEncodingTxtpb is the text protobuf message encoding.
	MessageEncodingTxtpb
	// MessageEncodingYAML is the YAML message encoding.
	MessageEncodingYAML

	useProtoNamesKey  = "use_proto_names"
	useEnumNumbersKey = "use_enum_numbers"
)

var (
	// MessageFormatsString is the string representation of all message formats.
	//
	// This does not include deprecated formats.
	MessageFormatsString = stringutil.SliceToString(messageFormatsNotDeprecated)
	// SourceDirFormatsString is the string representation of all source directory formats.
	// This includes all of the formats in SourceFormatsString except the protofile format.
	//
	// This does not include deprecated formats.
	SourceDirFormatsString = stringutil.SliceToString(sourceDirFormatsNotDeprecated)
	// SourceFormatsString is the string representation of all source formats.
	//
	// This does not include deprecated formats.
	SourceFormatsString = stringutil.SliceToString(sourceFormatsNotDeprecated)
	// ModuleFormatsString is the string representation of all module formats.
	//
	// Module formats are also source formats.
	//
	// This does not include deprecated formats.
	ModuleFormatsString = stringutil.SliceToString(moduleFormatsNotDeprecated)
	// SourceOrModuleFormatsString is the string representation of all source or module formats.
	//
	// This does not include deprecated formats.
	SourceOrModuleFormatsString = stringutil.SliceToString(sourceOrModuleFormatsNotDeprecated)
	// DirOrProtoFileFormats is the string representation of all dir or proto file formats.
	//
	// This does not include deprecated formats.
	DirOrProtoFileFormatsString = stringutil.SliceToString(dirOrProtoFileFormats)
	// AllFormatsString is the string representation of all formats.
	//
	// This does not include deprecated formats.
	AllFormatsString = stringutil.SliceToString(allFormatsNotDeprecated)

	// ErrModuleFormatDetectedForDirOrProtoFileRef is the error returned if a module is the
	// detected format in the DirOrProtoFileRefParser. We have a special heuristic to determine
	// if a path is a module or directory, and if a user specifies a suspected module, we want to error.
	ErrModuleFormatDetectedForDirOrProtoFileRef = errors.New("module format detected when parsing dir or proto file refs")
)

// MessageEncoding is the encoding of the message.
type MessageEncoding int

// Ref is an message file or source bucket reference.
type Ref interface {
	internalRef() internal.Ref
}

// MessageRef is an message file reference.
type MessageRef interface {
	Ref
	MessageEncoding() MessageEncoding
	// Path returns the path of the file.
	//
	// May be used for items such as YAML unmarshaling errors.
	Path() string
	// UseProtoNames only applies for MessageEncodingYAML at this time.
	UseProtoNames() bool
	// UseEnumNumbers only applies for MessageEncodingYAML at this time.
	UseEnumNumbers() bool
	IsNull() bool
	internalSingleRef() internal.SingleRef
}

// SourceOrModuleRef is a source bucket or module reference.
type SourceOrModuleRef interface {
	Ref
	isSourceOrModuleRef()
}

// DirOrProtoFileRef is a directory or proto file reference.
type DirOrProtoFileRef interface {
	isDirOrProtoFileRef()
}

// SourceRef is a source bucket reference.
type SourceRef interface {
	SourceOrModuleRef
	internalBucketRef() internal.BucketRef
}

// DirRef is a dir bucket reference.
type DirRef interface {
	SourceRef
	DirOrProtoFileRef
	DirPath() string
	internalDirRef() internal.DirRef
}

// ModuleRef is a module reference.
type ModuleRef interface {
	SourceOrModuleRef
	internalModuleRef() internal.ModuleRef
}

// ProtoFileRef is a proto file reference.
type ProtoFileRef interface {
	SourceRef
	DirOrProtoFileRef
	ProtoFilePath() string
	// True if the FileScheme is Stdio, Stdout, Stdin, or Null.
	IsDevPath() bool
	IncludePackageFiles() bool
	internalProtoFileRef() internal.ProtoFileRef
}

// MessageRefParser is an message ref parser for Buf.
type MessageRefParser interface {
	// GetMessageRef gets the reference for the message file.
	GetMessageRef(ctx context.Context, value string) (MessageRef, error)
	// GetMessageRefForInputConfig gets the reference for the message file.
	GetMessageRefForInputConfig(
		ctx context.Context,
		inputConfig bufconfig.InputConfig,
	) (MessageRef, error)
}

// SourceRefParser is a source ref parser for Buf.
type SourceRefParser interface {
	// GetSourceRef gets the reference for the source file.
	GetSourceRef(ctx context.Context, value string) (SourceRef, error)
	// GetSourceRef gets the reference for the source file.
	GetSourceRefForInputConfig(
		ctx context.Context,
		inputConfig bufconfig.InputConfig,
	) (SourceRef, error)
}

// DirRefParser is a dir ref parser for Buf.
type DirRefParser interface {
	// GetDirRef gets the reference for the value.
	//
	// The value cannot be stdin, stdout, or stderr.
	GetDirRef(ctx context.Context, value string) (DirRef, error)
	// GetDirRefForInputConfig gets the reference for the InputConfig.
	//
	// The input cannot be stdin, stdout, or stderr.
	GetDirRefForInputConfig(
		ctx context.Context,
		inputConfig bufconfig.InputConfig,
	) (DirRef, error)
}

// DirOrProtoFileRefParser is a dir or proto file ref parser for Buf.
type DirOrProtoFileRefParser interface {
	// GetDirOrProtoFileRef gets the reference for the value.
	//
	// The value cannot be stdin, stdout, or stderr.
	GetDirOrProtoFileRef(ctx context.Context, value string) (DirOrProtoFileRef, error)
	// GetDirOrProtoFileRefForInputConfig gets the reference for the InputConfig.
	//
	// The input cannot be stdin, stdout, or stderr.
	GetDirOrProtoFileRefForInputConfig(
		ctx context.Context,
		inputConfig bufconfig.InputConfig,
	) (DirOrProtoFileRef, error)
}

// ModuleRefParser is a source ref parser for Buf.
type ModuleRefParser interface {
	// GetModuleRef gets the reference for the source file.
	//
	// A module is a special type of source with additional properties.
	GetModuleRef(ctx context.Context, value string) (ModuleRef, error)
}

// SourceOrModuleRefParser is a source or module ref parser for Buf.
type SourceOrModuleRefParser interface {
	SourceRefParser
	ModuleRefParser

	// GetSourceOrModuleRef gets the reference for the message file or source bucket.
	GetSourceOrModuleRef(ctx context.Context, value string) (SourceOrModuleRef, error)
	// GetSourceOrModuleRefForInputConfig gets the reference for the message file or source bucket.
	GetSourceOrModuleRefForInputConfig(
		ctx context.Context,
		inputConfig bufconfig.InputConfig,
	) (SourceOrModuleRef, error)
}

// RefParser is a ref parser for Buf.
type RefParser interface {
	MessageRefParser
	SourceRefParser
	DirRefParser
	SourceOrModuleRefParser

	// TODO FUTURE: should this be renamed to GetRefForString?
	// GetRef gets the reference for the message file, source bucket, or module.
	GetRef(ctx context.Context, value string) (Ref, error)
	// GetRefForInputConfig gets the reference for the message file, source bucket, or module.
	GetRefForInputConfig(ctx context.Context, inputConfig bufconfig.InputConfig) (Ref, error)
}

// NewRefParser returns a new RefParser.
//
// This defaults to dir or module.
func NewRefParser(logger *zap.Logger) RefParser {
	return newRefParser(logger)
}

// NewMessageRefParser returns a new RefParser for messages only.
func NewMessageRefParser(logger *zap.Logger, options ...MessageRefParserOption) MessageRefParser {
	return newMessageRefParser(logger, options...)
}

// MessageRefParserOption is an option for a new MessageRefParser.
type MessageRefParserOption func(*messageRefParserOptions)

// MessageRefParserWithDefaultMessageEncoding says to use the default MessageEncoding.
//
// The default default is MessageEncodingBinpb.
func MessageRefParserWithDefaultMessageEncoding(defaultMessageEncoding MessageEncoding) MessageRefParserOption {
	return func(messageRefParserOptions *messageRefParserOptions) {
		messageRefParserOptions.defaultMessageEncoding = defaultMessageEncoding
	}
}

// NewSourceRefParser returns a new RefParser for sources only.
//
// This defaults to dir.
func NewSourceRefParser(logger *zap.Logger) SourceRefParser {
	return newSourceRefParser(logger)
}

// NewDirRefParser returns a new RefParser for dirs only.
func NewDirRefParser(logger *zap.Logger) DirRefParser {
	return newDirRefParser(logger)
}

// NewDirOrProtoFileRefParser returns a new RefParser for dirs only.
func NewDirOrProtoFileRefParser(logger *zap.Logger) DirOrProtoFileRefParser {
	return newDirOrProtoFileRefParser(logger)
}

// NewModuleRefParser returns a new RefParser for modules only.
func NewModuleRefParser(logger *zap.Logger) ModuleRefParser {
	return newModuleRefParser(logger)
}

// NewSourceOrModuleRefParser returns a new RefParser for sources or modules only.
//
// This defaults to dir or module.
func NewSourceOrModuleRefParser(logger *zap.Logger) SourceOrModuleRefParser {
	return newSourceOrModuleRefParser(logger)
}

// BucketExtender matches the internal type.
type BucketExtender internal.BucketExtender

// ReadBucketCloser matches the internal type.
type ReadBucketCloser internal.ReadBucketCloser

// ReadWriteBucket matches the internal type.
type ReadWriteBucket internal.ReadWriteBucket

// MessageReader is a message reader.
type MessageReader interface {
	// GetMessageFile gets the message file.
	//
	// The returned file will be uncompressed.
	GetMessageFile(
		ctx context.Context,
		container app.EnvStdinContainer,
		messageRef MessageRef,
	) (io.ReadCloser, error)
}

// SourceReader is a source reader.
type SourceReader interface {
	// GetSourceReadBucketCloser gets the source bucket.
	GetSourceReadBucketCloser(
		ctx context.Context,
		container app.EnvStdinContainer,
		sourceRef SourceRef,
		options ...GetReadBucketCloserOption,
	) (ReadBucketCloser, buftarget.BucketTargeting, error)
}

// GetReadBucketCloserOption is an option for a GetSourceReadBucketCloser call.
type GetReadBucketCloserOption func(*getReadBucketCloserOptions)

// GetReadBucketCloserCopyToInMemory says to copy the returned ReadBucketCloser to an
// in-memory ReadBucketCloser. This can be a performance optimization at the expense of memory.
func GetReadBucketCloserWithCopyToInMemory() GetReadBucketCloserOption {
	return func(getReadBucketCloserOptions *getReadBucketCloserOptions) {
		getReadBucketCloserOptions.copyToInMemory = true
	}
}

// GetReadBucketCloserWithNoSearch says to not search for buf.work.yamls or buf.yamls, instead just returning a bucket for the
// direct SourceRef or DirRef given.
//
// This is used for when the --config flag is specified.
func GetReadBucketCloserWithNoSearch() GetReadBucketCloserOption {
	return func(getReadBucketCloserOptions *getReadBucketCloserOptions) {
		getReadBucketCloserOptions.noSearch = true
	}
}

// GetReadBucketCloserWithTargetPaths sets the targets paths for bucket targeting information
// returned with the bucket.
func GetReadBucketCloserWithTargetPaths(targetPaths []string) GetReadBucketCloserOption {
	return func(getReadBucketCloserOptions *getReadBucketCloserOptions) {
		getReadBucketCloserOptions.targetPaths = targetPaths
	}
}

// GetReadBucketCloserWithTargetExcludePaths sets the target exclude paths for bucket targeting
// information returned with the bucket.
func GetReadBucketCloserWithTargetExcludePaths(targetExcludePaths []string) GetReadBucketCloserOption {
	return func(getReadBucketCloserOptions *getReadBucketCloserOptions) {
		getReadBucketCloserOptions.targetExcludePaths = targetExcludePaths
	}
}

// DirReader is a dir reader.
type DirReader interface {
	// GetDirReadWriteBucket gets the dir bucket.
	GetDirReadWriteBucket(
		ctx context.Context,
		container app.EnvStdinContainer,
		dirRef DirRef,
		options ...GetReadWriteBucketOption,
	) (ReadWriteBucket, buftarget.BucketTargeting, error)
}

// GetReadWriteBucketOption is an option for a GetDirReadWriteBucket call.
type GetReadWriteBucketOption func(*getReadWriteBucketOptions)

// GetReadWriteBucketWithNoSearch says to not search for buf.work.yamls or buf.yamls, instead just returning a bucket for the
// direct SourceRef or DirRef given.
//
// This is used for when the --config flag is specified.
func GetReadWriteBucketWithNoSearch() GetReadWriteBucketOption {
	return func(getReadWriteBucketOptions *getReadWriteBucketOptions) {
		getReadWriteBucketOptions.noSearch = true
	}
}

// GetReadWriteBucketWithTargetPaths sets the target paths for the bucket targeting information
// returned with the bucket.
func GetReadWriteBucketWithTargetPaths(targetPaths []string) GetReadWriteBucketOption {
	return func(getReadWriteBucketOptions *getReadWriteBucketOptions) {
		getReadWriteBucketOptions.targetPaths = targetPaths
	}
}

// GetReadWriteBucketWithTargetExcludePaths sets the target exclude paths for the bucket
// targeting information returned with the bucket.
func GetReadWriteBucketWithTargetExcludePaths(targetExcludePaths []string) GetReadWriteBucketOption {
	return func(getReadWriteBucketOptions *getReadWriteBucketOptions) {
		getReadWriteBucketOptions.targetExcludePaths = targetExcludePaths
	}
}

// ModuleFetcher is a module fetcher.
type ModuleFetcher interface {
	// GetModuleKey gets the ModuleKey.
	// Unresolved ModuleRef's are automatically resolved.
	GetModuleKey(
		ctx context.Context,
		container app.EnvStdinContainer,
		moduleRef ModuleRef,
	) (bufmodule.ModuleKey, error)
}

// Reader is a reader for Buf.
type Reader interface {
	MessageReader
	SourceReader
	DirReader
	ModuleFetcher
}

// NewReader returns a new Reader.
func NewReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
	gitCloner git.Cloner,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
) Reader {
	return newReader(
		logger,
		storageosProvider,
		httpClient,
		httpAuthenticator,
		gitCloner,
		moduleKeyProvider,
	)
}

// NewMessageReader returns a new MessageReader.
func NewMessageReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
	gitCloner git.Cloner,
) MessageReader {
	return newMessageReader(
		logger,
		storageosProvider,
		httpClient,
		httpAuthenticator,
		gitCloner,
	)
}

// NewSourceReader returns a new SourceReader.
func NewSourceReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
	gitCloner git.Cloner,
) SourceReader {
	return newSourceReader(
		logger,
		storageosProvider,
		httpClient,
		httpAuthenticator,
		gitCloner,
	)
}

// NewDirReader returns a new DirReader.
func NewDirReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) DirReader {
	return newDirReader(
		logger,
		storageosProvider,
	)
}

// NewModuleFetcher returns a new ModuleFetcher.
func NewModuleFetcher(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
) ModuleFetcher {
	return newModuleFetcher(
		logger,
		storageosProvider,
		moduleKeyProvider,
	)
}

// Writer is a writer for Buf.
type Writer interface {
	// PutMessageFile puts the message file.
	PutMessageFile(
		ctx context.Context,
		container app.EnvStdoutContainer,
		messageRef MessageRef,
	) (io.WriteCloser, error)
}

// NewWriter returns a new Writer.
func NewWriter(
	logger *zap.Logger,
) Writer {
	return newWriter(
		logger,
	)
}

// ProtoFileWriter is a writer of proto files.
type ProtoFileWriter interface {
	// PutProtoFile puts the proto file.
	PutProtoFile(
		ctx context.Context,
		container app.EnvStdoutContainer,
		protoFileRef ProtoFileRef,
	) (io.WriteCloser, error)
}

// NewProtoFileWriter returns a new ProtoFileWriter.
func NewProtoFileWriter(
	logger *zap.Logger,
) ProtoFileWriter {
	return newProtoFileWriter(
		logger,
	)
}

// GetInputConfigForString returns the input config for the input string.
func GetInputConfigForString(
	ctx context.Context,
	refParser RefParser,
	value string,
) (bufconfig.InputConfig, error) {
	ref, err := refParser.GetRef(ctx, value)
	if err != nil {
		return nil, err
	}
	switch t := ref.(type) {
	case MessageRef:
		switch t.MessageEncoding() {
		case MessageEncodingBinpb:
			return bufconfig.NewBinaryImageInputConfig(
				t.Path(),
				t.internalSingleRef().CompressionType().String(),
			)
		case MessageEncodingJSON:
			return bufconfig.NewJSONImageInputConfig(
				t.Path(),
				t.internalSingleRef().CompressionType().String(),
			)
		case MessageEncodingTxtpb:
			return bufconfig.NewTextImageInputConfig(
				t.Path(),
				t.internalSingleRef().CompressionType().String(),
			)
		case MessageEncodingYAML:
			return bufconfig.NewYAMLImageInputConfig(
				t.Path(),
				t.internalSingleRef().CompressionType().String(),
			)
		default:
			return nil, fmt.Errorf("unknown encoding: %v", t.MessageEncoding())
		}
	}
	return internal.GetInputConfigForRef(ref.internalRef(), value)
}

type getReadBucketCloserOptions struct {
	noSearch           bool
	copyToInMemory     bool
	targetPaths        []string
	targetExcludePaths []string
}

func newGetReadBucketCloserOptions() *getReadBucketCloserOptions {
	return &getReadBucketCloserOptions{}
}

type getReadWriteBucketOptions struct {
	noSearch           bool
	targetPaths        []string
	targetExcludePaths []string
}

func newGetReadWriteBucketOptions() *getReadWriteBucketOptions {
	return &getReadWriteBucketOptions{}
}
