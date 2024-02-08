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

// Package storagearchive implements archive utilities.
package storagearchive

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"strings"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
	"github.com/klauspost/compress/zip"
	"go.uber.org/multierr"
)

var (
	// ErrFileSizeLimit is returned when file read limit is reached.
	//
	// See [WithMaxFileSizeUntarOption]
	ErrFileSizeLimit = errors.New("file size exceeded read limit")
)

// Tar tars the given bucket to the writer.
//
// Only regular files are added to the writer.
// All files are written as 0644.
func Tar(
	ctx context.Context,
	readBucket storage.ReadBucket,
	writer io.Writer,
) (retErr error) {
	tarWriter := tar.NewWriter(writer)
	defer func() {
		retErr = multierr.Append(retErr, tarWriter.Close())
	}()
	return storage.WalkReadObjects(
		ctx,
		readBucket,
		"",
		func(readObject storage.ReadObject) error {
			data, err := io.ReadAll(readObject)
			if err != nil {
				return err
			}
			if err := tarWriter.WriteHeader(
				&tar.Header{
					Typeflag: tar.TypeReg,
					Name:     readObject.Path(),
					Size:     int64(len(data)),
					// If we ever use this outside of testing, we will want to do something about this
					Mode: 0644,
				},
			); err != nil {
				return err
			}
			_, err = tarWriter.Write(data)
			return err
		},
	)
}

// Untar untars the given tar archive from the reader into the bucket.
//
// Only regular files are added to the bucket.
//
// Paths from the tar archive will be mapped before adding to the bucket.
// Mapper can be nil.
// StripComponents happens before the mapper.
func Untar(
	ctx context.Context,
	reader io.Reader,
	writeBucket storage.WriteBucket,
	mapper storage.Mapper,
	stripComponentCount uint32,
	opts ...UntarOption,
) error {
	options := &untarOptions{
		maxFileSize: math.MaxInt64,
	}
	for _, opt := range opts {
		opt.applyUntar(options)
	}
	tarReader := tar.NewReader(reader)
	walkChecker := storageutil.NewWalkChecker()
	for tarHeader, err := tarReader.Next(); err != io.EOF; tarHeader, err = tarReader.Next() {
		if err != nil {
			return err
		}
		if err := walkChecker.Check(ctx); err != nil {
			return err
		}
		if tarHeader.Size < 0 {
			return fmt.Errorf("invalid size for tar file %s: %d", tarHeader.Name, tarHeader.Size)
		}
		if isAppleExtendedAttributesFile(tarHeader.FileInfo()) {
			continue
		}
		path, ok, err := unmapArchivePath(tarHeader.Name, mapper, stripComponentCount)
		if err != nil {
			return err
		}
		if !ok || !tarHeader.FileInfo().Mode().IsRegular() {
			continue
		}
		if tarHeader.Size > options.maxFileSize {
			return fmt.Errorf("%w %s:%d", ErrFileSizeLimit, tarHeader.Name, tarHeader.Size)
		}
		if err := storage.CopyReader(ctx, writeBucket, tarReader, path); err != nil {
			return err
		}
	}
	return nil
}

// UntarOption is an option for [Untar].
type UntarOption interface {
	applyUntar(*untarOptions)
}

// WithMaxFileSizeUntarOption returns an option that limits the maximum size
func WithMaxFileSizeUntarOption(size int) UntarOption {
	return &withMaxFileSizeUntarOption{maxFileSize: int64(size)}
}

// Zip zips the given bucket to the writer.
//
// Only regular files are added to the writer.
func Zip(
	ctx context.Context,
	readBucket storage.ReadBucket,
	writer io.Writer,
	compressed bool,
) (retErr error) {
	zipWriter := zip.NewWriter(writer)
	defer func() {
		retErr = multierr.Append(retErr, zipWriter.Close())
	}()
	return storage.WalkReadObjects(
		ctx,
		readBucket,
		"",
		func(readObject storage.ReadObject) error {
			method := zip.Store
			if compressed {
				method = zip.Deflate
			}
			header := &zip.FileHeader{
				Name:   readObject.Path(),
				Method: method,
			}
			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return err
			}
			_, err = io.Copy(writer, readObject)
			return err
		},
	)
}

// Unzip unzips the given zip archive from the reader into the bucket.
//
// Only regular files are added to the bucket.
//
// Paths from the zip archive will be mapped before adding to the bucket.
// Mapper can be nil.
// StripComponents happens before the mapper.
func Unzip(
	ctx context.Context,
	readerAt io.ReaderAt,
	size int64,
	writeBucket storage.WriteBucket,
	mapper storage.Mapper,
	stripComponentCount uint32,
) error {
	if size < 0 {
		return fmt.Errorf("unknown size to unzip: %d", int(size))
	}
	if size == 0 {
		return nil
	}
	zipReader, err := zip.NewReader(readerAt, size)
	if err != nil {
		return err
	}
	walkChecker := storageutil.NewWalkChecker()
	// reads can be done concurrently in the future
	for _, zipFile := range zipReader.File {
		if err := walkChecker.Check(ctx); err != nil {
			return err
		}
		path, ok, err := unmapArchivePath(zipFile.Name, mapper, stripComponentCount)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if isAppleExtendedAttributesFile(zipFile.FileInfo()) {
			continue
		}
		if zipFile.FileInfo().Mode().IsRegular() {
			if err := copyZipFile(ctx, writeBucket, zipFile, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func isAppleExtendedAttributesFile(fileInfo fs.FileInfo) bool {
	// On macOS, .tar archives created with libarchive will contain additional
	// files with a prefix of "._" if there are files with extended attributes
	// and copyfile is enabled.
	// Archive Utility.app has a similar behavior when creating .zip archives,
	// except they are placed under a separate MACOSX directory tree.
	// Here, both are handled by just ignoring all files with a "._" prefix.
	// This is a reasonable compromise because files that live in a Module
	// (.proto files, configuration files such as buf.yaml, README files) are
	// almost never prefixed with ._, and fixing this issue in this manner
	// outweighs the slight incorrectness.
	return strings.HasPrefix(fileInfo.Name(), "._")
}

func copyZipFile(
	ctx context.Context,
	writeBucket storage.WriteBucket,
	zipFile *zip.File,
	path string,
) (retErr error) {
	readCloser, err := zipFile.Open()
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	return storage.CopyReader(ctx, writeBucket, readCloser, path)
}

func unmapArchivePath(
	archivePath string,
	mapper storage.Mapper,
	stripComponentCount uint32,
) (string, bool, error) {
	if archivePath == "" {
		return "", false, errors.New("empty archive file name")
	}
	fullPath, err := normalpath.NormalizeAndValidate(archivePath)
	if err != nil {
		return "", false, err
	}
	if fullPath == "." {
		return "", false, nil
	}
	fullPath, ok := normalpath.StripComponents(fullPath, stripComponentCount)
	if !ok {
		return "", false, nil
	}
	if mapper != nil {
		return mapper.UnmapFullPath(fullPath)
	}
	return fullPath, true, nil
}
