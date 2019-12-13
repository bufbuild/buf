package bufbuild

import (
	"context"
	"sort"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type handler struct {
	logger        *zap.Logger
	buildProvider Provider
	buildRunner   Runner
}

func newHandler(
	logger *zap.Logger,
	buildProvider Provider,
	buildRunner Runner,
) *handler {
	return &handler{
		logger:        logger.Named("bufbuild"),
		buildProvider: buildProvider,
		buildRunner:   buildRunner,
	}
}

func (h *handler) BuildImage(
	ctx context.Context,
	bucket storage.ReadBucket,
	roots []string,
	excludes []string,
	specificRealFilePaths []string,
	specificRealFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (_ bufpb.Image, _ ProtoFilePathResolver, _ []*analysis.Annotation, retErr error) {
	var copyToMemory bool
	var protoFileSet ProtoFileSet
	var err error
	if len(specificRealFilePaths) > 0 {
		copyToMemory = false
		protoFileSet, err = h.buildProvider.GetProtoFileSetForRealFilePaths(
			ctx,
			bucket,
			roots,
			specificRealFilePaths,
			specificRealFilePathsAllowNotExist,
		)
	} else {
		copyToMemory = true
		protoFileSet, err = h.buildProvider.GetProtoFileSetForBucket(
			ctx,
			bucket,
			roots,
			excludes,
		)
	}
	if err != nil {
		return nil, nil, nil, err
	}

	if copyToMemory {
		memBucket, err := h.copyToMemory(ctx, bucket, protoFileSet)
		if err != nil {
			return nil, nil, nil, err
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

	image, annotations, err := h.buildRunner.Run(
		ctx,
		bucket,
		protoFileSet,
		includeImports,
		includeSourceInfo,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(annotations) > 0 {
		if err := FixAnnotationFilenames(protoFileSet, annotations); err != nil {
			return nil, nil, nil, err
		}
		return nil, nil, annotations, nil
	}
	return image, protoFileSet, nil, nil
}

func (h *handler) ListFiles(
	ctx context.Context,
	bucket storage.ReadBucket,
	roots []string,
	excludes []string,
) ([]string, error) {
	protoFileSet, err := h.buildProvider.GetProtoFileSetForBucket(ctx, bucket, roots, excludes)
	if err != nil {
		return nil, err
	}
	files := protoFileSet.RealFilePaths()
	// The files are in the order of the root file paths, we want to sort them for output.
	sort.Strings(files)
	return files, nil
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
