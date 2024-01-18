// Copyright 2020-2024 Buf Technologies, Inc.
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

package protostatstorage

import (
	"context"
	"io"

	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

type fileWalker struct {
	readBucket storage.ReadBucket
}

func newFileWalker(readBucket storage.ReadBucket) *fileWalker {
	return &fileWalker{
		readBucket: storage.MapReadBucket(
			readBucket,
			storage.MatchPathExt(".proto"),
		),
	}
}

func (f *fileWalker) Walk(ctx context.Context, fu func(io.Reader) error) error {
	return f.readBucket.Walk(
		ctx,
		"",
		func(objectInfo storage.ObjectInfo) (retErr error) {
			readObjectCloser, err := f.readBucket.Get(ctx, objectInfo.Path())
			if err != nil {
				return err
			}
			defer func() {
				retErr = multierr.Append(retErr, readObjectCloser.Close())
			}()
			return fu(readObjectCloser)
		},
	)
}
