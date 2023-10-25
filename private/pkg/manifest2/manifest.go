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

package manifest2

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

type manifest struct {
	pathToDigest map[string]Digest
	sortedPaths  []string
}

func newManifest(pathToDigest map[string]Digest) (*manifest, error) {
	sortedPaths := make([]string, 0, len(pathToDigest))
	for path := range pathToDigest {
		if path == "" {
			return nil, errors.New("empty path in Manifest construction")
		}
		normalizedPath, err := normalpath.NormalizeAndValidate(path)
		if err != nil {
			return nil, fmt.Errorf("normalization error in Manifest construction: %w", err)
		}
		if path != normalizedPath {
			return nil, fmt.Errorf("path %q did not equal normalized path %q in Manifest construction", path, normalizedPath)
		}
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)
	return &manifest{
		pathToDigest: pathToDigest,
		sortedPaths:  sortedPaths,
	}, nil
}

func (m *manifest) ForEach(f func(path string, digest Digest) error) error {
	for _, path := range m.sortedPaths {
		if err := f(path, m.pathToDigest[path]); err != nil {
			return err
		}
	}
	return nil
}

func (m *manifest) String() string {
	buffer := bytes.NewBuffer(nil)
	for _, path := range m.sortedPaths {
		_, _ = buffer.WriteString(m.pathToDigest[path].String())
		_, _ = buffer.WriteString("  ")
		_, _ = buffer.WriteString(path)
		_, _ = buffer.WriteRune('\n')
	}
	return buffer.String()
}

func (*manifest) isManifest() {}
