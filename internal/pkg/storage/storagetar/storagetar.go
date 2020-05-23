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
	"archive/tar"
	"context"
	"io"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
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
	transformer := normalpath.NewTransformer(options...)
	tarReader := tar.NewReader(reader)
	for tarHeader, err := tarReader.Next(); err != io.EOF; tarHeader, err = tarReader.Next() {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		path, err := normalpath.NormalizeAndValidate(tarHeader.Name)
		if err != nil {
			return err
		}
		if path == "." {
			continue
		}
		path, ok := transformer.Transform(path)
		if !ok {
			continue
		}
		if tarHeader.FileInfo().Mode().IsRegular() {
			writeObject, err := readWriteBucket.Put(ctx, path, uint32(tarHeader.Size))
			if err != nil {
				return err
			}
			_, writeErr := io.Copy(writeObject, tarReader)
			if err := writeObject.Close(); err != nil {
				return err
			}
			if writeErr != nil {
				return writeErr
			}
		}
	}
	return nil
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
	options ...normalpath.TransformerOption,
) (retErr error) {
	transformer := normalpath.NewTransformer(options...)
	tarWriter := tar.NewWriter(writer)
	defer func() {
		retErr = multierr.Append(retErr, tarWriter.Close())
	}()
	return readBucket.Walk(
		ctx,
		"",
		func(path string) error {
			newPath, ok := transformer.Transform(path)
			if !ok {
				return nil
			}
			readObject, err := readBucket.Get(ctx, path)
			if err != nil {
				return err
			}
			if err := tarWriter.WriteHeader(
				&tar.Header{
					Typeflag: tar.TypeReg,
					Name:     newPath,
					Size:     int64(readObject.Size()),
					// If we ever use this outside of testing, we will want to do something about this
					Mode: 0644,
				},
			); err != nil {
				return multierr.Append(err, readObject.Close())
			}
			_, err = io.Copy(tarWriter, readObject)
			return multierr.Append(err, readObject.Close())
		},
	)
}
