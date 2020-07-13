// Copyright 2020 Buf Technologies, Inc.
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

package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/diff"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

// Diff does a diff of the ReadBuckets.
func Diff(
	ctx context.Context,
	one ReadBucket,
	two ReadBucket,
	oneBucketName string,
	twoBucketName string,
) ([]byte, error) {
	onePaths, err := AllPaths(ctx, one, "")
	if err != nil {
		return nil, err
	}
	twoPaths, err := AllPaths(ctx, two, "")
	if err != nil {
		return nil, err
	}
	onePathMap := stringutil.SliceToMap(onePaths)
	twoPathMap := stringutil.SliceToMap(twoPaths)
	var onlyInOne []string
	buffer := bytes.NewBuffer(nil)
	for _, path := range onePaths {
		if _, ok := twoPathMap[path]; ok {
			oneData, err := ReadPath(ctx, one, path)
			if err != nil {
				return nil, err
			}
			twoData, err := ReadPath(ctx, two, path)
			if err != nil {
				return nil, err
			}
			diffData, err := diff.Diff(
				ctx,
				oneData,
				twoData,
				normalpath.Join(oneBucketName, path),
				normalpath.Join(twoBucketName, path),
				false,
			)
			if err != nil {
				return nil, err
			}
			if len(diffData) > 0 {
				_, _ = buffer.Write(diffData)
			}
		} else {
			onlyInOne = append(onlyInOne, path)
		}
	}
	for _, path := range onlyInOne {
		_, _ = buffer.WriteString(fmt.Sprintf("Only in %s: %s\n", oneBucketName, path))
	}
	for _, path := range twoPaths {
		if _, ok := onePathMap[path]; !ok {
			_, _ = buffer.WriteString(fmt.Sprintf("Only in %s: %s\n", twoBucketName, path))
		}
	}
	return buffer.Bytes(), nil
}
