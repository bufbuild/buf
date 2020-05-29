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
	"fmt"
	"sort"

	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"go.uber.org/zap"
)

type fileRefProvider struct {
	logger *zap.Logger
}

func newFileRefProvider(logger *zap.Logger) *fileRefProvider {
	return &fileRefProvider{
		logger: logger,
	}
}

func (p *fileRefProvider) GetAllFileRefs(
	ctx context.Context,
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
	roots []string,
	excludes []string,
) ([]bufimage.FileRef, error) {
	defer instrument.Start(p.logger, "get_all_file_fefs").End()

	roots, excludes, err := normalizeAndValidateRootsExcludes(roots, excludes)
	if err != nil {
		return nil, err
	}

	// map from file path relative to root, to file path relative to bucket, to raw file ref
	// need all this info for duplicate error messages
	rootRelFilePathToFullRelFilePathToRawFileRef := make(map[string]map[string]*rawFileRef)
	for _, root := range roots {
		if walkErr := readBucket.Walk(
			ctx,
			root,
			// all fullRelFilePath values are already normalized and validated
			func(fullRelFilePath string) error {
				if normalpath.Ext(fullRelFilePath) != ".proto" {
					return nil
				}
				// get relative to root
				rootRelFilePath, err := normalpath.Rel(root, fullRelFilePath)
				if err != nil {
					return err
				}
				// just in case
				rootRelFilePath, err = normalpath.NormalizeAndValidate(rootRelFilePath)
				if err != nil {
					return err
				}
				fullRelFilePathToRawFileRef, ok := rootRelFilePathToFullRelFilePathToRawFileRef[rootRelFilePath]
				if !ok {
					fullRelFilePathToRawFileRef = make(map[string]*rawFileRef)
					rootRelFilePathToFullRelFilePathToRawFileRef[rootRelFilePath] = fullRelFilePathToRawFileRef
				}
				fullRelFilePathToRawFileRef[fullRelFilePath] = newRawFileRef(rootRelFilePath, fullRelFilePath, root)
				return nil
			},
		); walkErr != nil {
			return nil, walkErr
		}
	}

	rawFileRefs := make([]*rawFileRef, 0, len(rootRelFilePathToFullRelFilePathToRawFileRef))
	for rootRelFilePath, fullRelFilePathToRawFileRef := range rootRelFilePathToFullRelFilePathToRawFileRef {
		fullRelFilePaths := make([]string, 0, len(fullRelFilePathToRawFileRef))
		iRawFileRefs := make([]*rawFileRef, 0, len(fullRelFilePathToRawFileRef))
		for fullRelFilePath, rawFileRef := range fullRelFilePathToRawFileRef {
			fullRelFilePaths = append(fullRelFilePaths, fullRelFilePath)
			iRawFileRefs = append(iRawFileRefs, rawFileRef)
		}
		switch len(fullRelFilePaths) {
		case 0:
			// we expect to always have at least one value, this is a system error
			return nil, fmt.Errorf("no real file path for file path %q", rootRelFilePath)
		case 1:
			rawFileRefs = append(rawFileRefs, iRawFileRefs[0])
		default:
			sort.Strings(fullRelFilePaths)
			return nil, fmt.Errorf("file with path %s is within multiple roots at %v", rootRelFilePath, fullRelFilePaths)
		}
	}

	if len(excludes) == 0 {
		if len(rawFileRefs) == 0 {
			return nil, errors.New("no input files found that match roots")
		}
		return getSortedFileRefs(rawFileRefs, externalPathResolver)
	}

	filteredRawFileRefs := make([]*rawFileRef, 0, len(rawFileRefs))
	excludeMap := stringutil.SliceToMap(excludes)
	for _, rawFileRef := range rawFileRefs {
		if !normalpath.MapContainsMatch(excludeMap, normalpath.Dir(normalpath.Join(rawFileRef.FullRelFilePath))) {
			filteredRawFileRefs = append(filteredRawFileRefs, rawFileRef)
		}
	}
	if len(filteredRawFileRefs) == 0 {
		return nil, errors.New("no input files found that match roots and excludes")
	}
	return getSortedFileRefs(filteredRawFileRefs, externalPathResolver)
}

func (p *fileRefProvider) GetFileRefsForExternalFilePaths(
	ctx context.Context,
	readBucket storage.ReadBucket,
	pathResolver bufpath.PathResolver,
	roots []string,
	externalFilePaths []string,
	options ...GetFileRefsForExternalFilePathsOption,
) ([]bufimage.FileRef, error) {
	getFileRefsForExternalFilePathsOptions := newGetFileRefsForExternalFilePathsOptions()
	for _, option := range options {
		option(getFileRefsForExternalFilePathsOptions)
	}
	return p.getFileRefsForExternalFilePaths(
		ctx,
		readBucket,
		pathResolver,
		roots,
		externalFilePaths,
		getFileRefsForExternalFilePathsOptions.allowNotExist,
	)
}

func (p *fileRefProvider) getFileRefsForExternalFilePaths(
	ctx context.Context,
	readBucket storage.ReadBucket,
	pathResolver bufpath.PathResolver,
	roots []string,
	externalFilePaths []string,
	allowNotExist bool,
) ([]bufimage.FileRef, error) {
	defer instrument.Start(p.logger, "get_file_refs_for_external_file_paths").End()

	roots, err := normalizeAndValidateRoots(roots)
	if err != nil {
		return nil, err
	}
	fullRelFilePathMap := make(map[string]struct{}, len(externalFilePaths))
	for _, externalFilePath := range externalFilePaths {
		fullRelFilePath, err := pathResolver.ExternalPathToRelPath(externalFilePath)
		if err != nil {
			return nil, err
		}
		if _, ok := fullRelFilePathMap[fullRelFilePath]; ok {
			return nil, fmt.Errorf("duplicate file path %s", fullRelFilePath)
		}
		// check that the file exists primarily
		if _, err := readBucket.Stat(ctx, fullRelFilePath); err != nil {
			if !storage.IsNotExist(err) {
				return nil, err
			}
			if !allowNotExist {
				return nil, err
			}
		} else {
			fullRelFilePathMap[fullRelFilePath] = struct{}{}
		}
	}

	rootMap := stringutil.SliceToMap(roots)
	rawFileRefs := make([]*rawFileRef, 0, len(fullRelFilePathMap))
	rootRelFilePathToFullRelFilePath := make(map[string]string, len(fullRelFilePathMap))
	for fullRelFilePath := range fullRelFilePathMap {
		matchingRootMap := normalpath.MapMatches(rootMap, fullRelFilePath)
		matchingRoots := make([]string, 0, len(matchingRootMap))
		for matchingRoot := range matchingRootMap {
			matchingRoots = append(matchingRoots, matchingRoot)
		}
		switch len(matchingRoots) {
		case 0:
			return nil, fmt.Errorf("file %s is not within any root %v", fullRelFilePath, roots)
		case 1:
			rootDirPath := matchingRoots[0]
			rootRelFilePath, err := normalpath.Rel(rootDirPath, fullRelFilePath)
			if err != nil {
				return nil, err
			}
			// just in case
			// return system error as this would be an issue
			rootRelFilePath, err = normalpath.NormalizeAndValidate(rootRelFilePath)
			if err != nil {
				// This is a system error
				return nil, err
			}
			if otherFullRelFilePath, ok := rootRelFilePathToFullRelFilePath[rootRelFilePath]; ok {
				return nil, fmt.Errorf("file with path %s is within another root as %s at %s", fullRelFilePath, rootRelFilePath, otherFullRelFilePath)
			}
			rawFileRefs = append(rawFileRefs, newRawFileRef(rootRelFilePath, fullRelFilePath, rootDirPath))
			rootRelFilePathToFullRelFilePath[rootRelFilePath] = fullRelFilePath
		default:
			sort.Strings(matchingRoots)
			// this should probably never happen due to how we are doing this with matching roots but just in case
			return nil, fmt.Errorf("file with path %s is within multiple roots at %v", fullRelFilePath, matchingRoots)
		}
	}

	if len(rawFileRefs) == 0 {
		return nil, errors.New("no input files specified")
	}
	return getSortedFileRefs(rawFileRefs, pathResolver)
}

func getSortedFileRefs(
	rawFileRefs []*rawFileRef,
	externalPathResolver bufpath.ExternalPathResolver,
) ([]bufimage.FileRef, error) {
	sort.Slice(rawFileRefs, func(i int, j int) bool { return rawFileRefs[i].RootRelFilePath < rawFileRefs[j].RootRelFilePath })

	fileRefs := make([]bufimage.FileRef, len(rawFileRefs))
	for i, rawFileRef := range rawFileRefs {
		fileRef, err := bufimage.NewFileRef(rawFileRef.RootRelFilePath, rawFileRef.RootDirPath, externalPathResolver)
		if err != nil {
			return nil, err
		}
		fileRefs[i] = fileRef
	}
	return fileRefs, nil
}

type rawFileRef struct {
	RootRelFilePath string
	FullRelFilePath string
	RootDirPath     string
}

func newRawFileRef(
	rootRelFilePath string,
	fullRelFilePath string,
	rootDirPath string,
) *rawFileRef {
	return &rawFileRef{
		RootRelFilePath: rootRelFilePath,
		FullRelFilePath: fullRelFilePath,
		RootDirPath:     rootDirPath,
	}
}

type getFileRefsForExternalFilePathsOptions struct {
	allowNotExist bool
}

func newGetFileRefsForExternalFilePathsOptions() *getFileRefsForExternalFilePathsOptions {
	return &getFileRefsForExternalFilePathsOptions{}
}
