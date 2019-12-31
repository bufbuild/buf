package bufbuild

import (
	"context"
	"time"

	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/analysis"
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
}

func newHandler(
	logger *zap.Logger,
) *handler {
	return &handler{
		logger:   logger.Named("bufbuild"),
		provider: newProvider(logger),
		runner:   newRunner(logger),
	}
}

func (h *handler) Build(
	ctx context.Context,
	bucket storage.ReadBucket,
	protoFileSet ProtoFileSet,
	options BuildOptions,
) (_ *imagev1beta1.Image, _ []*analysis.Annotation, retErr error) {
	if options.CopyToMemory {
		memBucket, err := h.copyToMemory(ctx, bucket, protoFileSet)
		if err != nil {
			return nil, nil, err
		}
		if memBucket != nil {
			bucket = memBucket
			defer func() {
				retErr = multierr.Append(retErr, memBucket.Close())
			}()
		}
	} else {
		h.logger.Debug("no_copy_to_memory_set")
	}

	image, annotations, err := h.runner.Run(
		ctx,
		bucket,
		protoFileSet,
		options.IncludeImports,
		options.IncludeSourceInfo,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(annotations) > 0 {
		if err := FixAnnotationFilenames(protoFileSet, annotations); err != nil {
			return nil, nil, err
		}
		return nil, annotations, nil
	}
	return image, nil, nil
}

func (h *handler) Files(
	ctx context.Context,
	bucket storage.ReadBucket,
	options FilesOptions,
) (ProtoFileSet, error) {
	if len(options.SpecificRealFilePaths) > 0 {
		return h.provider.GetProtoFileSetForRealFilePaths(
			ctx,
			bucket,
			options.Roots,
			options.SpecificRealFilePaths,
			options.SpecificRealFilePathsAllowNotExist,
		)
	}
	return h.provider.GetProtoFileSetForBucket(
		ctx,
		bucket,
		options.Roots,
		options.Excludes,
	)
}

// copyToMemory copies the bucket to memory.
//
// If the bucket was already in memory, this returns nil.
// Returns error on system error.
func (h *handler) copyToMemory(
	ctx context.Context,
	bucket storage.ReadBucket,
	protoFileSet ProtoFileSet,
) (storage.ReadBucket, error) {
	start := time.Now()

	if bucket.Type() == storagemem.BucketType {
		h.logger.Debug("already_in_memory")
		return nil, nil
	}

	memBucket := storagemem.NewBucket()
	count, err := storageutil.CopyPaths(
		ctx,
		bucket,
		memBucket,
		protoFileSet.RealFilePaths()...,
	)
	if err != nil {
		return nil, multierr.Append(err, memBucket.Close())
	}
	h.logger.Debug(
		"copy_to_memory",
		zap.Int("num_files", count),
		zap.Duration("duration", time.Since(start)),
	)
	return memBucket, nil
}
