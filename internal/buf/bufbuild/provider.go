package bufbuild

import (
	"context"
	"sort"

	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/logutil"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"go.uber.org/zap"
)

type provider struct {
	logger *zap.Logger
}

func newProvider(logger *zap.Logger) *provider {
	return &provider{
		logger: logger.Named("build"),
	}
}

func (p *provider) GetProtoFileSetForBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
	config *Config,
) (ProtoFileSet, error) {
	defer logutil.Defer(p.logger, "get_proto_file_set_for_bucket")()

	if len(config.Roots) == 0 {
		return nil, errs.NewInvalidArgument("no roots specified")
	}

	// map from file path relative to root, to all actual file paths
	rootFilePathToRealFilePathMap := make(map[string]map[string]struct{})
	for _, root := range config.Roots {
		if walkErr := bucket.Walk(
			ctx,
			root,
			// all realFilePath values are already normalized and validated
			func(realFilePath string) error {
				if storagepath.Ext(realFilePath) != ".proto" {
					return nil
				}
				// get relative to root
				rootFilePath, err := storagepath.Rel(root, realFilePath)
				if err != nil {
					return err
				}
				// just in case
				// return system error as this would be an issue
				rootFilePath, err = storagepath.NormalizeAndValidate(rootFilePath)
				if err != nil {
					return errs.NewInternal(err.Error())
				}
				realFilePathMap, ok := rootFilePathToRealFilePathMap[rootFilePath]
				if !ok {
					realFilePathMap = make(map[string]struct{})
					rootFilePathToRealFilePathMap[rootFilePath] = realFilePathMap
				}
				realFilePathMap[realFilePath] = struct{}{}
				return nil
			},
		); walkErr != nil {
			return nil, walkErr
		}
	}

	rootFilePathToRealFilePath := make(map[string]string, len(rootFilePathToRealFilePathMap))
	for rootFilePath, realFilePathMap := range rootFilePathToRealFilePathMap {
		realFilePaths := make([]string, 0, len(realFilePathMap))
		for realFilePath := range realFilePathMap {
			realFilePaths = append(realFilePaths, realFilePath)
		}
		switch len(realFilePaths) {
		case 0:
			// we expect to always have at least one value, this is a system error
			return nil, errs.NewInternalf("no real file path for file path %q", rootFilePath)
		case 1:
			rootFilePathToRealFilePath[rootFilePath] = realFilePaths[0]
		default:
			sort.Strings(realFilePaths)
			return nil, errs.NewInvalidArgumentf("file with path %s is within multiple roots at %v", rootFilePath, realFilePaths)
		}
	}

	if len(config.Excludes) == 0 {
		if len(rootFilePathToRealFilePath) == 0 {
			return nil, errs.NewInvalidArgument("no input files found that match roots")
		}
		return newProtoFileSet(config.Roots, rootFilePathToRealFilePath)
	}

	filteredRootFilePathToRealFilePath := make(map[string]string, len(rootFilePathToRealFilePath))
	excludeMap := stringutil.SliceToMap(config.Excludes)
	for rootFilePath, realFilePath := range rootFilePathToRealFilePath {
		if !storagepath.MapContainsMatch(excludeMap, storagepath.Dir(realFilePath)) {
			filteredRootFilePathToRealFilePath[rootFilePath] = realFilePath
		}
	}
	if len(filteredRootFilePathToRealFilePath) == 0 {
		return nil, errs.NewInvalidArgument("no input files found that match roots and excludes")
	}
	return newProtoFileSet(config.Roots, filteredRootFilePathToRealFilePath)
}

func (p *provider) GetProtoFileSetForRealFilePaths(
	ctx context.Context,
	bucket storage.ReadBucket,
	config *Config,
	realFilePaths []string,
	allowNotExist bool,
) (ProtoFileSet, error) {
	defer logutil.Defer(p.logger, "get_proto_file_set_for_real_file_paths")()

	if len(config.Roots) == 0 {
		return nil, errs.NewInvalidArgument("no roots specified")
	}

	normalizedRealFilePaths := make(map[string]struct{}, len(realFilePaths))
	for _, realFilePath := range realFilePaths {
		normalizedRealFilePath, err := storagepath.NormalizeAndValidate(realFilePath)
		if err != nil {
			return nil, err
		}
		if _, ok := normalizedRealFilePaths[normalizedRealFilePath]; ok {
			return nil, errs.NewInvalidArgumentf("duplicate normalized file path %s", normalizedRealFilePath)
		}
		// check that the file exists primarily
		if _, err := bucket.Stat(ctx, normalizedRealFilePath); err != nil {
			if !storage.IsNotExist(err) {
				return nil, err
			}
			if !allowNotExist {
				return nil, errs.NewInvalidArgument(err.Error())
			}
		} else {
			normalizedRealFilePaths[normalizedRealFilePath] = struct{}{}
		}
	}

	rootMap := stringutil.SliceToMap(config.Roots)
	rootFilePathToRealFilePath := make(map[string]string, len(normalizedRealFilePaths))
	for normalizedRealFilePath := range normalizedRealFilePaths {
		matchingRootMap := storagepath.MapMatches(rootMap, normalizedRealFilePath)
		matchingRoots := make([]string, 0, len(matchingRootMap))
		for matchingRoot := range matchingRootMap {
			matchingRoots = append(matchingRoots, matchingRoot)
		}
		switch len(matchingRoots) {
		case 0:
			return nil, errs.NewInvalidArgumentf("file %s is not within any root %v", normalizedRealFilePath, config.Roots)
		case 1:
			normalizedRootFilePath, err := storagepath.Rel(matchingRoots[0], normalizedRealFilePath)
			if err != nil {
				return nil, err
			}
			// just in case
			// return system error as this would be an issue
			normalizedRootFilePath, err = storagepath.NormalizeAndValidate(normalizedRootFilePath)
			if err != nil {
				// This is a system error
				return nil, errs.NewInternal(err.Error())
			}
			if otherRealFilePath, ok := rootFilePathToRealFilePath[normalizedRootFilePath]; ok {
				return nil, errs.NewInvalidArgumentf("file with path %s is within another root as %s at %s", normalizedRealFilePath, normalizedRootFilePath, otherRealFilePath)
			}
			rootFilePathToRealFilePath[normalizedRootFilePath] = normalizedRealFilePath
		default:
			sort.Strings(matchingRoots)
			// this should probably never happen due to how we are doing this with matching roots but just in case
			return nil, errs.NewInvalidArgumentf("file with path %s is within multiple roots at %v", normalizedRealFilePath, matchingRoots)
		}
	}

	if len(rootFilePathToRealFilePath) == 0 {
		return nil, errs.NewInvalidArgument("no input files specified")
	}
	return newProtoFileSet(config.Roots, rootFilePathToRealFilePath)
}
