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
	"context"

	"github.com/bufbuild/buf/internal/pkg/storage/internal"
)

// MultiReadBucket takes the union of the ReadBuckets.
//
// If no readBuckets are given, this returns a no-op ReadBucket.
// If one readBucket is given, this returns the original ReadBucket.
// Otherwise, this returns a ReadBucket that will get from all buckets.
//
// This expects and validates that no paths overlap between the ReadBuckets.
// This assumes that buckets are logically unique.
func MultiReadBucket(readBuckets ...ReadBucket) ReadBucket {
	switch len(readBuckets) {
	case 0:
		return nopReadBucket{}
	case 1:
		return readBuckets[0]
	default:
		return newMultiReadBucket(readBuckets)
	}
}

type multiReadBucket struct {
	delegates []ReadBucket
}

func newMultiReadBucket(
	delegates []ReadBucket,
) *multiReadBucket {
	return &multiReadBucket{
		delegates: delegates,
	}
}

func (m *multiReadBucket) Get(ctx context.Context, path string) (ReadObjectCloser, error) {
	_, delegateIndex, err := m.getObjectInfoAndDelegateIndex(ctx, path)
	if err != nil {
		return nil, err
	}
	return m.delegates[delegateIndex].Get(ctx, path)
}

func (m *multiReadBucket) Stat(ctx context.Context, path string) (ObjectInfo, error) {
	objectInfo, _, err := m.getObjectInfoAndDelegateIndex(ctx, path)
	return objectInfo, err
}

func (m *multiReadBucket) Walk(ctx context.Context, prefix string, f func(ObjectInfo) error) error {
	seenPathToExternalPath := make(map[string]string)
	for _, delegate := range m.delegates {
		if err := delegate.Walk(
			ctx,
			prefix,
			func(objectInfo ObjectInfo) error {
				path := objectInfo.Path()
				externalPath := objectInfo.ExternalPath()
				if existingExternalPath, ok := seenPathToExternalPath[path]; ok {
					// this does not return all paths that are matching, unlike Get and Stat
					// we do not want to continue iterating, as calling Walk on the same path could cause errors downstream
					// as callers expect a single call per path.
					return NewErrExistsMultipleLocations(path, existingExternalPath, externalPath)
				}
				seenPathToExternalPath[path] = externalPath
				return f(objectInfo)
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func (m *multiReadBucket) getObjectInfoAndDelegateIndex(
	ctx context.Context,
	path string,
) (ObjectInfo, int, error) {
	var objectInfos []ObjectInfo
	var delegateIndex int
	for i, delegate := range m.delegates {
		objectInfo, err := delegate.Stat(ctx, path)
		if err != nil {
			if IsNotExist(err) {
				continue
			}
			return nil, 0, err
		}
		objectInfos = append(objectInfos, objectInfo)
		delegateIndex = i
	}
	switch len(objectInfos) {
	case 0:
		return nil, 0, NewErrNotExist(path)
	case 1:
		return objectInfos[0], delegateIndex, nil
	default:
		externalPaths := make([]string, len(objectInfos))
		for i, objectInfo := range objectInfos {
			externalPaths[i] = objectInfo.ExternalPath()
		}
		return nil, 0, NewErrExistsMultipleLocations(path, externalPaths...)
	}
}

type nopReadBucket struct{}

func (nopReadBucket) Get(ctx context.Context, path string) (ReadObjectCloser, error) {
	return nil, nopGetStat(path)
}

func (nopReadBucket) Stat(ctx context.Context, path string) (ObjectInfo, error) {
	return nil, nopGetStat(path)
}

func (nopReadBucket) Walk(ctx context.Context, prefix string, f func(ObjectInfo) error) error {
	_, err := internal.ValidatePrefix(prefix)
	return err
}

func nopGetStat(path string) error {
	path, err := internal.ValidatePath(path)
	if err != nil {
		return err
	}
	return NewErrNotExist(path)
}
