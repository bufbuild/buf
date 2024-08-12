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

package storage

import (
	"context"
	"io/fs"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// FilterReadBucket filters the ReadBucket.
//
// If the Matchers are empty, the original ReadBucket is returned.
// If there is more than one Matcher, the Matchers are anded together.
func FilterReadBucket(readBucket ReadBucket, matchers ...Matcher) ReadBucket {
	if len(matchers) == 0 {
		return readBucket
	}
	return newFilterReadBucketCloser(readBucket, nil, MatchAnd(matchers...))
}

// FilterReadBucketCloser filters the ReadBucketCloser.
//
// If the Matchers are empty, the original ReadBucketCloser is returned.
// If there is more than one Matcher, the Matchers are anded together.
func FilterReadBucketCloser(readBucketCloser ReadBucketCloser, matchers ...Matcher) ReadBucketCloser {
	if len(matchers) == 0 {
		return readBucketCloser
	}
	return newFilterReadBucketCloser(readBucketCloser, readBucketCloser.Close, MatchAnd(matchers...))
}

type filterReadBucketCloser struct {
	delegate  ReadBucket
	closeFunc func() error
	matcher   Matcher
}

func newFilterReadBucketCloser(
	delegate ReadBucket,
	closeFunc func() error,
	matcher Matcher,
) *filterReadBucketCloser {
	return &filterReadBucketCloser{
		delegate:  delegate,
		closeFunc: closeFunc,
		matcher:   matcher,
	}
}

func (r *filterReadBucketCloser) Get(ctx context.Context, path string) (ReadObjectCloser, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if !r.matcher.MatchPath(path) {
		return nil, &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
	}
	return r.delegate.Get(ctx, path)
}

func (r *filterReadBucketCloser) Stat(ctx context.Context, path string) (ObjectInfo, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if !r.matcher.MatchPath(path) {
		return nil, &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
	}
	return r.delegate.Stat(ctx, path)
}

func (r *filterReadBucketCloser) Walk(ctx context.Context, prefix string, f func(ObjectInfo) error) error {
	prefix, err := normalpath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
	return r.delegate.Walk(
		ctx,
		prefix,
		func(objectInfo ObjectInfo) error {
			if !r.matcher.MatchPath(objectInfo.Path()) {
				return nil
			}
			return f(objectInfo)
		},
	)
}

func (r *filterReadBucketCloser) Close() error {
	if r.closeFunc != nil {
		return r.closeFunc()
	}
	return nil
}
