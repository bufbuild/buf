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

package storagetar

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
)

func doUntar(
	ctx context.Context,
	reader io.Reader,
	readWriteBucket storage.ReadWriteBucket,
	gzipped bool,
	options []normalpath.TransformerOption,
) error {
	transformer := normalpath.NewTransformer(options...)
	var err error
	if gzipped {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return err
		}
	}
	tarReader := tar.NewReader(reader)
	fileCount := 0
	for tarHeader, err := tarReader.Next(); err != io.EOF; tarHeader, err = tarReader.Next() {
		if err != nil {
			return err
		}
		fileCount++
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

func doTar(
	ctx context.Context,
	writer io.Writer,
	readBucket storage.ReadBucket,
	prefix string,
	gzipped bool,
	options []normalpath.TransformerOption,
) (retErr error) {
	transformer := normalpath.NewTransformer(options...)
	if gzipped {
		gzipWriter := gzip.NewWriter(writer)
		defer func() {
			retErr = multierr.Append(retErr, gzipWriter.Close())
		}()
		writer = gzipWriter
	}
	tarWriter := tar.NewWriter(writer)
	defer func() {
		retErr = multierr.Append(retErr, tarWriter.Close())
	}()
	return readBucket.Walk(
		ctx,
		prefix,
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
