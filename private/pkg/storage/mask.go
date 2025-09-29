// Copyright 2020-2025 Buf Technologies, Inc.
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
	"io/fs"
	"slices"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// MaskReadBucket creates a ReadBucket that masks the include and exclude prefixes,
// with optimized walking that only traverses include prefixes.
//
// If includePrefixes is empty, all paths are included (no prefix filtering on walk).
// If excludePrefixes is provided, those paths are excluded from results.
// If both includePrefixes and excludePrefixes are empty, the original ReadBucket is returned.
func MaskReadBucket(readBucket ReadBucket, includePrefixes, excludePrefixes []string) (ReadBucket, error) {
	if len(includePrefixes) == 0 && len(excludePrefixes) == 0 {
		return readBucket, nil
	}
	return newMaskReadBucketCloser(readBucket, nil, includePrefixes, excludePrefixes)
}

// MaskReadBucketCloser creates a ReadBucketCloser that masks using include and exclude prefixes,
// with optimized walking that only traverses include prefixes.
//
// If includePrefixes is empty, all paths are included (no prefix filtering on walk).
// If excludePrefixes is provided, those paths are excluded from results.
// If both includePrefixes and excludePrefixes are empty, the original ReadBucketCloser is returned.
func MaskReadBucketCloser(readBucketCloser ReadBucketCloser, includePrefixes, excludePrefixes []string) (ReadBucketCloser, error) {
	if len(includePrefixes) == 0 && len(excludePrefixes) == 0 {
		return readBucketCloser, nil
	}
	return newMaskReadBucketCloser(readBucketCloser, readBucketCloser.Close, includePrefixes, excludePrefixes)
}

type maskReadBucketCloser struct {
	delegate        ReadBucket
	closeFunc       func() error
	includePrefixes []string
	excludePrefixes []string
}

func newMaskReadBucketCloser(
	delegate ReadBucket,
	closeFunc func() error,
	includePrefixes, excludePrefixes []string,
) (*maskReadBucketCloser, error) {
	normalizedIncludes, err := normalizeValidateAndCompactPrefixes(includePrefixes)
	if err != nil {
		return nil, err
	}
	normalizedExcludes, err := normalizeValidateAndCompactPrefixes(excludePrefixes)
	if err != nil {
		return nil, err
	}
	return &maskReadBucketCloser{
		delegate:        delegate,
		closeFunc:       closeFunc,
		includePrefixes: normalizedIncludes,
		excludePrefixes: normalizedExcludes,
	}, nil
}

func (r *maskReadBucketCloser) Get(ctx context.Context, path string) (ReadObjectCloser, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if !r.matchPath(path) {
		return nil, &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
	}
	return r.delegate.Get(ctx, path)
}

func (r *maskReadBucketCloser) Stat(ctx context.Context, path string) (ObjectInfo, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if !r.matchPath(path) {
		return nil, &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
	}
	return r.delegate.Stat(ctx, path)
}

func (r *maskReadBucketCloser) Walk(ctx context.Context, prefix string, f func(ObjectInfo) error) error {
	prefix, err := normalpath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
	for _, excludePrefix := range r.excludePrefixes {
		isChild := normalpath.EqualsOrContainsPath(excludePrefix, prefix, normalpath.Relative)
		if isChild {
			// The requested prefix is under an exclude prefix, so nothing to walk.
			return nil
		}
	}
	walkFunc := func(objectInfo ObjectInfo) error {
		if !r.matchPath(objectInfo.Path()) {
			return nil
		}
		return f(objectInfo)
	}
	if len(r.includePrefixes) == 0 {
		// No include prefixes, so walk normally.
		return r.delegate.Walk(ctx, prefix, walkFunc)
	}
	// Find all include prefixes under the requests root prefix.
	var effectivePrefixes []string
	for _, includePrefix := range r.includePrefixes {
		isParent := normalpath.EqualsOrContainsPath(includePrefix, prefix, normalpath.Relative)
		if isParent {
			// The requested prefix is under an include prefix, so walk normally.
			return r.delegate.Walk(ctx, prefix, walkFunc)
		}
		isChild := normalpath.EqualsOrContainsPath(prefix, includePrefix, normalpath.Relative)
		if isChild {
			effectivePrefixes = append(effectivePrefixes, includePrefix)
		}
	}
	// Walk each effective prefix that is a child of the requested prefix.
	// The effective prefixes are sorted and compacted on creation of the Bucket,
	// so no need to sort or compact here.
	for _, effectivePrefix := range effectivePrefixes {
		if err := r.delegate.Walk(ctx, effectivePrefix, walkFunc); err != nil {
			return err
		}
	}
	return nil
}

func (r *maskReadBucketCloser) Close() error {
	if r.closeFunc != nil {
		return r.closeFunc()
	}
	return nil
}

// matchPath checks if a path matches the include/exclude criteria
func (r *maskReadBucketCloser) matchPath(path string) bool {
	// Check excludes first (if any exclude matches, reject the path)
	for _, excludePrefix := range r.excludePrefixes {
		// Check if the exclude prefix contains the path (path is under exclude prefix)
		if normalpath.EqualsOrContainsPath(excludePrefix, path, normalpath.Relative) {
			return false
		}
	}
	// If no include prefixes, accept all paths (that weren't excluded)
	if len(r.includePrefixes) == 0 {
		return true
	}
	// Check includes (at least one include must match)
	for _, includePrefix := range r.includePrefixes {
		// Check if the include prefix contains the path (path is under include prefix)
		if normalpath.EqualsOrContainsPath(includePrefix, path, normalpath.Relative) {
			return true
		}
	}
	return false
}

// normalizeValidateAndCompactPrefixes normalizes, validates, and compacts a list of path prefixes.
// It removes redundant child prefixes that are already covered by parent prefixes.
// For example, ["foo", "foo/v1", "foo/v1/v2"] becomes ["foo"].
func normalizeValidateAndCompactPrefixes(prefixes []string) ([]string, error) {
	if len(prefixes) == 0 {
		return nil, nil
	}
	var normalized []string
	for _, prefix := range prefixes {
		normalizedPrefix, err := normalpath.NormalizeAndValidate(prefix)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, normalizedPrefix)
	}
	slices.Sort(normalized)
	// Remove redundant child prefixes that are covered by parent prefixes.
	// For example, ["bar", "foo", "foo/v1", "foo/v1/v2"] becomes ["bar", "foo"].
	reduced := normalized[:1]
	for _, prefix := range normalized[1:] {
		if !normalpath.EqualsOrContainsPath(reduced[len(reduced)-1], prefix, normalpath.Relative) {
			reduced = append(reduced, prefix)
		}
	}
	return reduced, nil
}
