package bufmodule

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

const (
	// licenseFilePath is the path of the license file within a Module.
	licenseFilePath = "LICENSE"
)

var (
	// orderedDocFilePaths are the potential documentation file paths for a Module.
	//
	// When creating a Module from a Bucket, we check the file paths buf.md, README.md, and README.markdown
	// to exist, in that order. The first one to exist is chosen as the documentation file that is considered
	// part of the Module, and any others are discarded.
	orderedDocFilePaths = []string{
		"buf.md",
		"README.md",
		"README.markdown",
	}

	docFilePathMap map[string]struct{}
)

func init() {
	docFilePathMap = stringutil.SliceToMap(orderedDocFilePaths)
}

func getDocFilePathForStorageReadBucket(ctx context.Context, bucket storage.ReadBucket) string {
	for _, docFilePath := range orderedDocFilePaths {
		if _, err := bucket.Stat(ctx, docFilePath); err == nil {
			return docFilePath
		}
	}
	return ""
}

func getDocFilePathForModuleReadBucket(ctx context.Context, bucket ModuleReadBucket) string {
	for _, docFilePath := range orderedDocFilePaths {
		if _, err := bucket.StatFileInfo(ctx, docFilePath); err == nil {
			return docFilePath
		}
	}
	return ""
}

func allDocFilePathsExcept(exceptDocFilePath string) []string {
	docFilePaths := make([]string, 0, len(orderedDocFilePaths)-1)
	for _, docFilePath := range orderedDocFilePaths {
		if docFilePath != exceptDocFilePath {
			docFilePaths = append(docFilePaths, docFilePath)
		}
	}
	return docFilePaths
}
