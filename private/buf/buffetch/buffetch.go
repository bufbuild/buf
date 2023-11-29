// Copyright 2020-2023 Buf Technologies, Inc.
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
	"io"
	"net/http"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
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
	// AllFormatsString is the string representation of all formats.
	//
	// This does not include deprecated formats.
	AllFormatsString = stringutil.SliceToString(allFormatsNotDeprecated)
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

// SourceRef is a source bucket reference.
type SourceRef interface {
	SourceOrModuleRef
	internalBucketRef() internal.BucketRef
}

// DirRef is a dir bucket reference.
type DirRef interface {
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
	IncludePackageFiles() bool
	internalProtoFileRef() internal.ProtoFileRef
}

// MessageRefParser is an message ref parser for Buf.
type MessageRefParser interface {
	// GetMessageRef gets the reference for the message file.
	GetMessageRef(ctx context.Context, value string) (MessageRef, error)
}

// SourceRefParser is a source ref parser for Buf.
type SourceRefParser interface {
	// GetSourceRef gets the reference for the source file.
	GetSourceRef(ctx context.Context, value string) (SourceRef, error)
}

// DirRefParser is a dif ref parser for Buf.
type DirRefParser interface {
	// GetDirRef gets the reference for the source file.
	GetDirRef(ctx context.Context, value string) (DirRef, error)
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
}

// RefParser is a ref parser for Buf.
type RefParser interface {
	MessageRefParser
	SourceRefParser
	DirRefParser
	SourceOrModuleRefParser

	// GetRef gets the reference for the message file, source bucket, or module.
	GetRef(ctx context.Context, value string) (Ref, error)
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
	) (ReadBucketCloser, error)
}

// DirReader is a dir reader.
type DirReader interface {
	// GetDirReadWriteBucket gets the dir bucket.
	GetDirReadWriteBucket(
		ctx context.Context,
		container app.EnvStdinContainer,
		dirRef DirRef,
	) (ReadWriteBucket, error)
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
