package bufbuild

import (
	"context"
	"errors"
	"time"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type handler struct {
	logger   *zap.Logger
	provider *provider
	runner   *runner

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
	handler.provider = newProvider(logger)
	handler.runner = newRunner(logger, handler.parallelism)
	return handler
}

func (h *handler) Build(
	ctx context.Context,
	readBucket storage.ReadBucket,
	protoFileSet ProtoFileSet,
	options BuildOptions,
) (_ *imagev1beta1.Image, _ []*filev1beta1.FileAnnotation, retErr error) {
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

	image, fileAnnotations, err := h.runner.Run(
		ctx,
		readBucket,
		protoFileSet,
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
	return image, nil, nil
}

func (h *handler) GetProtoFileSet(
	ctx context.Context,
	readBucket storage.ReadBucket,
	options GetProtoFileSetOptions,
) (ProtoFileSet, error) {
	return h.provider.GetProtoFileSetForReadBucket(
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
	return h.provider.GetProtoFileSetForRealFilePaths(
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
	start := time.Now()

	if readBucket.Info().InMemory() {
		h.logger.Debug("already_in_memory")
		return nil, nil
	}

	memReadWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	count, err := storageutil.CopyPaths(
		ctx,
		readBucket,
		memReadWriteBucketCloser,
		protoFileSet.RealFilePaths()...,
	)
	if err != nil {
		return nil, multierr.Append(err, memReadWriteBucketCloser.Close())
	}
	h.logger.Debug(
		"copy_to_memory",
		zap.Int("num_files", count),
		zap.Duration("duration", time.Since(start)),
	)
	return memReadWriteBucketCloser, nil
}
