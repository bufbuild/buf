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

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

// Store provides disk-backed WKT buckets.
type Store interface {
	// GetBucket gets a disk-backed WKT bucket.
	//
	// LocalPaths will be present on all files within the bucket.
	GetBucket(ctx context.Context) (storage.ReadBucket, error)
}

// NewStore returns a new Store for the given cache bucket.
//
// It is assumed that the Store has complete control of the bucket.
func NewStore(
	logger *zap.Logger,
	runner command.Runner,
	bucket storage.ReadWriteBucket,
) Store {
	return newStore(logger, runner, bucket)
}
