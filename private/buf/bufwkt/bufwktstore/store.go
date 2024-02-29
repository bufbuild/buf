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

package bufwktstore

import (
	"context"

	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

type store struct {
	logger *zap.Logger
	runner command.Runner
	bucket storage.ReadWriteBucket
}

func newStore(
	logger *zap.Logger,
	runner command.Runner,
	bucket storage.ReadWriteBucket,
) *store {
	return &store{
		logger: logger,
		runner: runner,
		bucket: bucket,
	}
}

func (s *store) GetBucket(ctx context.Context) (storage.ReadBucket, error) {
	wktBucket := storage.MapReadWriteBucket(s.bucket, storage.MapOnPrefix(datawkt.Version))

	isEmpty, err := storage.IsEmpty(ctx, wktBucket, "")
	if err != nil {
		return nil, err
	}
	if isEmpty {
		if err := copyToBucket(ctx, wktBucket); err != nil {
			return nil, err
		}
	} else {
		diff, err := storage.DiffBytes(ctx, s.runner, datawkt.ReadBucket, wktBucket)
		if err != nil {
			return nil, err
		}
		if len(diff) > 0 {
			if err := deleteBucket(ctx, wktBucket); err != nil {
				return nil, err
			}
			if err := copyToBucket(ctx, wktBucket); err != nil {
				return nil, err
			}
		}
	}

	return storage.StripReadBucketExternalPaths(wktBucket), nil
}

func copyToBucket(ctx context.Context, wktBucket storage.WriteBucket) error {
	_, err := storage.Copy(ctx, datawkt.ReadBucket, wktBucket)
	return err
}

func deleteBucket(ctx context.Context, wktBucket storage.WriteBucket) error {
	return wktBucket.DeleteAll(ctx, "")
}
