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

package bufbuild

import (
	"context"
	"errors"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type handler struct {
	logger               *zap.Logger
	protoFileSetProvider *protoFileSetProvider
	builder              *builder

	parallelism               int
	copyToMemoryFileThreshold int
}

func newHandler(
	logger *zap.Logger,
	options ...HandlerOption,
) *handler {
	handler := &handler{
		logger:                    logger.Named("bufbuild"),
		parallelism:               DefaultParallelism,
		copyToMemoryFileThreshold: DefaultCopyToMemoryFileThreshold,
	}
	for _, option := range options {
		option(handler)
	}
	handler.protoFileSetProvider = newProtoFileSetProvider(logger)
	handler.builder = newBuilder(logger, handler.parallelism)
	return handler
}

func (h *handler) Build(
	ctx context.Context,
	readBucket storage.ReadBucket,
	protoFileSet ProtoFileSet,
	options BuildOptions,
) (_ *BuildResult, _ []*filev1beta1.FileAnnotation, retErr error) {
	if h.copyToMemoryFileThreshold > 0 && protoFileSet.Size() >= h.copyToMemoryFileThreshold {
		memReadWriteBucketCloser, err := h.copyToMemory(ctx, readBucket, protoFileSet)
		if err != nil {
			return nil, nil, err
		}
		if memReadWriteBucketCloser != nil {
			readBucket = memReadWriteBucketCloser
			defer func() {
				retErr = multierr.Append(retErr, memReadWriteBucketCloser.Close())
			}()
		}
	} else {
		h.logger.Debug("no_copy_to_memory_set")
	}

	buildResult, fileAnnotations, err := h.builder.Build(
		ctx,
		readBucket,
		protoFileSet.Roots(),
		protoFileSet.RootFilePaths(),
		options.IncludeImports,
		options.IncludeSourceInfo,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(fileAnnotations) > 0 {
		if err := FixFileAnnotationPaths(protoFileSet, fileAnnotations...); err != nil {
			return nil, nil, err
		}
		return nil, fileAnnotations, nil
	}
	return buildResult, nil, nil
}

func (h *handler) GetProtoFileSet(
	ctx context.Context,
	readBucket storage.ReadBucket,
	options GetProtoFileSetOptions,
) (ProtoFileSet, error) {
	return h.protoFileSetProvider.GetProtoFileSetForReadBucket(
		ctx,
		readBucket,
		options.Roots,
		options.Excludes,
	)
}

func (h *handler) GetProtoFileSetForFiles(
	ctx context.Context,
	readBucket storage.ReadBucket,
	realFilePaths []string,
	options GetProtoFileSetForFilesOptions,
) (ProtoFileSet, error) {
	if len(realFilePaths) == 0 {
		return nil, errors.New("no input files")
	}
	return h.protoFileSetProvider.GetProtoFileSetForRealFilePaths(
		ctx,
		readBucket,
		options.Roots,
		realFilePaths,
		options.AllowNotExist,
	)
}

// copyToMemory copies the bucket to memory.
//
// If the bucket was already in memory, this returns nil.
// Returns error on system error.
func (h *handler) copyToMemory(
	ctx context.Context,
	readBucket storage.ReadBucket,
	protoFileSet ProtoFileSet,
) (storage.ReadWriteBucketCloser, error) {
	if readBucket.Info().InMemory() {
		h.logger.Debug("already_in_memory")
		return nil, nil
	}
	timer := instrument.Start(h.logger, "copy_to_memory")
	memReadWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	count, err := storageutil.CopyPaths(
		ctx,
		readBucket,
		memReadWriteBucketCloser,
		// note we are not copying the configuration file here
		protoFileSet.RealFilePaths(),
	)
	if err != nil {
		return nil, multierr.Append(err, memReadWriteBucketCloser.Close())
	}
	timer.End(zap.Int("num_files", count))
	return memReadWriteBucketCloser, nil
}
