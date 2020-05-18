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

// Package storagetar implements tar utilities.
package storagetar

import (
	"context"
	"io"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
)

// Untar untars the given tar archive from the reader into the bucket.
//
// Only regular files are added to the bucket.
//
// Paths from the tar archive will be transformed before adding to the bucket.
func Untar(
	ctx context.Context,
	reader io.Reader,
	readWriteBucket storage.ReadWriteBucket,
	options ...normalpath.TransformerOption,
) error {
	return doUntar(ctx, reader, readWriteBucket, false, options)
}

// Untargz untars the given targz archive from the reader into the bucket.
//
// Only regular files are added to the bucket.
//
// Paths from the targz archive will be transformed before adding to the bucket.
func Untargz(
	ctx context.Context,
	reader io.Reader,
	readWriteBucket storage.ReadWriteBucket,
	options ...normalpath.TransformerOption,
) error {
	return doUntar(ctx, reader, readWriteBucket, true, options)
}

// Tar tars the given bucket to the writer.
//
// Only regular files are added to the writer.
// All files are written as 0644.
//
// Paths from the bucket will be transformed before adding to the writer.
func Tar(
	ctx context.Context,
	writer io.Writer,
	readBucket storage.ReadBucket,
	prefix string,
	options ...normalpath.TransformerOption,
) error {
	return doTar(ctx, writer, readBucket, prefix, false, options)
}

// Targz tars and gzips the given bucket to the writer.
//
// Only regular files are added to the writer.
// All files are written as 0644.
//
// Paths from the bucket will be transformed before adding to the writer.
func Targz(
	ctx context.Context,
	writer io.Writer,
	readBucket storage.ReadBucket,
	prefix string,
	options ...normalpath.TransformerOption,
) error {
	return doTar(ctx, writer, readBucket, prefix, true, options)
}
