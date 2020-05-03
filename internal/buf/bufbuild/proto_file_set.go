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
	"errors"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
)

type protoFileSet struct {
	roots                      []string
	rootFilePathToRealFilePath map[string]string

	realFilePathToRootFilePath map[string]string
	sortedRootRealFilePaths    []*rootRealFilePath
}

// It is expected that:
//
// - roots are normalized and validated (relative)
// - rootFilePathToRealFilePath paths are normalized and validated (relative)
// - every realFilePath is contained in a root
// - the mapping is 1-1
//
// However, if the mapping is not 1-1, this returns system error.
func newProtoFileSet(roots []string, rootFilePathToRealFilePath map[string]string) (*protoFileSet, error) {
	realFilePathToRootFilePath := make(map[string]string, len(rootFilePathToRealFilePath))
	rootRealFilePaths := make([]*rootRealFilePath, 0, len(rootFilePathToRealFilePath))
	for rootFilePath, realFilePath := range rootFilePathToRealFilePath {
		if _, ok := realFilePathToRootFilePath[realFilePath]; ok {
			return nil, fmt.Errorf("real file path %q passed with duplicate root file path %q", realFilePath, rootFilePath)
		}
		realFilePathToRootFilePath[realFilePath] = rootFilePath
		rootRealFilePaths = append(
			rootRealFilePaths,
			&rootRealFilePath{
				rootFilePath: rootFilePath,
				realFilePath: realFilePath,
			},
		)
	}
	if len(rootFilePathToRealFilePath) != len(realFilePathToRootFilePath) ||
		len(rootFilePathToRealFilePath) != len(rootRealFilePaths) {
		return nil, errors.New("inconsistent count of real to root file paths")
	}
	sort.Slice(
		rootRealFilePaths,
		func(i int, j int) bool {
			return rootRealFilePaths[i].rootFilePath < rootRealFilePaths[j].rootFilePath
		},
	)
	return &protoFileSet{
		roots:                      roots,
		rootFilePathToRealFilePath: rootFilePathToRealFilePath,
		realFilePathToRootFilePath: realFilePathToRootFilePath,
		sortedRootRealFilePaths:    rootRealFilePaths,
	}, nil
}

func (s *protoFileSet) Roots() []string {
	l := make([]string, len(s.roots))
	for i, root := range s.roots {
		l[i] = root
	}
	return l
}

func (s *protoFileSet) RootFilePaths() []string {
	l := make([]string, len(s.sortedRootRealFilePaths))
	for i, rootRealFilePath := range s.sortedRootRealFilePaths {
		l[i] = rootRealFilePath.rootFilePath
	}
	return l
}

func (s *protoFileSet) RealFilePaths() []string {
	l := make([]string, len(s.sortedRootRealFilePaths))
	for i, rootRealFilePath := range s.sortedRootRealFilePaths {
		l[i] = rootRealFilePath.realFilePath
	}
	return l
}

func (s *protoFileSet) Size() int {
	return len(s.sortedRootRealFilePaths)
}

func (s *protoFileSet) GetRootFilePath(realFilePath string) (string, error) {
	if realFilePath == "" {
		return "", errors.New("file path empty")
	}
	realFilePath, err := storagepath.NormalizeAndValidate(realFilePath)
	if err != nil {
		return "", err
	}
	rootFilePath, ok := s.realFilePathToRootFilePath[realFilePath]
	if !ok {
		return "", nil
	}
	return rootFilePath, nil
}

func (s *protoFileSet) GetRealFilePath(rootFilePath string) (string, error) {
	if rootFilePath == "" {
		return "", errors.New("file path empty")
	}
	rootFilePath, err := storagepath.NormalizeAndValidate(rootFilePath)
	if err != nil {
		return "", err
	}
	realFilePath, ok := s.rootFilePathToRealFilePath[rootFilePath]
	if !ok {
		return "", nil
	}
	return realFilePath, nil
}

type rootRealFilePath struct {
	rootFilePath string
	realFilePath string
}
