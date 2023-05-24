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

package git

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/filepathextended"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

type tagIterator struct {
	gitDirPath   string
	objectReader ObjectReader
}

func newTagIterator(
	gitDirPath string,
	objectReader ObjectReader,
) (TagIterator, error) {
	gitDirPath = normalpath.Unnormalize(gitDirPath)
	if err := validateDirPathExists(gitDirPath); err != nil {
		return nil, err
	}
	gitDirPath, err := filepath.Abs(gitDirPath)
	if err != nil {
		return nil, err
	}
	return &tagIterator{
		gitDirPath:   gitDirPath,
		objectReader: objectReader,
	}, nil
}

func (r *tagIterator) ForEachTag(f func(string, Hash) error) error {
	dir := path.Join(r.gitDirPath, "refs", "tags")
	return filepathextended.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		tagName, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		tagName = normalpath.Normalize(tagName)
		hashBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hashBytes = bytes.TrimSuffix(hashBytes, []byte{'\n'})
		hash, err := parseHashFromHex(string(hashBytes))
		if err != nil {
			return err
		}
		// Tags are either annotated or lightweight. Depending on the type,
		// they are stored differently. First, we try to load the tag
		// as an annnotated tag. If this fails, we try a commit.
		// Finally, we fail.
		tag, err := r.objectReader.Tag(hash)
		if err == nil {
			f(tagName, tag.Commit())
			return nil
		}
		if !errors.Is(err, errObjectTypeMismatch) {
			return err
		}
		_, err = r.objectReader.Commit(hash)
		if err == nil {
			f(tagName, hash)
			return nil
		}
		if !errors.Is(err, errObjectTypeMismatch) {
			return err
		}
		return fmt.Errorf(
			"failed to determine target of tag %q; it is neither a tag nor a commit",
			tagName,
		)
	})
}
