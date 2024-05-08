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
	options ...UntarOption,
) error {
	untarOptions := newUntarOptions()
	for _, option := range options {
		option(untarOptions)
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
		path, ok, err := unmapArchivePath(tarHeader.Name, untarOptions.filePathMatcher, untarOptions.stripComponentCount)
		if err != nil {
			return err
		}
		if !ok || !tarHeader.FileInfo().Mode().IsRegular() {
			continue
		}
		if untarOptions.maxFileSize != 0 && tarHeader.Size > untarOptions.maxFileSize {
			return fmt.Errorf("%w %s:%d", ErrFileSizeLimit, tarHeader.Name, tarHeader.Size)
		}
		if err := storage.CopyReader(ctx, writeBucket, tarReader, path); err != nil {
			return err
		}
	}
	return nil
}

// UntarOption is an option for Untar.
type UntarOption func(*untarOptions)

// UntarWithMaxFileSize returns a new UntarOption that limits the maximum file size.
//
// The default is to have no limit.
func UntarWithMaxFileSize(maxFileSize int64) UntarOption {
	return func(untarOptions *untarOptions) {
		untarOptions.maxFileSize = maxFileSize
	}
}

// UntarWithStripComponentCount returns a new UntarOption that strips the specified number of components.
func UntarWithStripComponentCount(stripComponentCount uint32) UntarOption {
	return func(untarOptions *untarOptions) {
		untarOptions.stripComponentCount = stripComponentCount
	}
}

// UntarWithFilePathMatcher returns a new UntarOption that will only write a given file to the
// bucket if the function returns true on the normalized file path.
//
// The matcher will be applied after components are stripped.
func UntarWithFilePathMatcher(filePathMatcher func(string) bool) UntarOption {
	return func(untarOptions *untarOptions) {
		untarOptions.filePathMatcher = filePathMatcher
	}
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
	options ...UnzipOption,
) error {
	unzipOptions := newUnzipOptions()
	for _, option := range options {
		option(unzipOptions)
	}
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
		path, ok, err := unmapArchivePath(zipFile.Name, unzipOptions.filePathMatcher, unzipOptions.stripComponentCount)
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

// UnzipOption is an option for Unzip.
type UnzipOption func(*unzipOptions)

// UnzipWithStripComponentCount returns a new UnzipOption that strips the specified number of components.
func UnzipWithStripComponentCount(stripComponentCount uint32) UnzipOption {
	return func(unzipOptions *unzipOptions) {
		unzipOptions.stripComponentCount = stripComponentCount
	}
}

// UnzipWithFilePathMatcher returns a new UnzipOption that will only write a given file to the
// bucket if the function returns true on the normalized file path.
//
// The matcher will be applied after components are stripped.
func UnzipWithFilePathMatcher(filePathMatcher func(string) bool) UnzipOption {
	return func(unzipOptions *unzipOptions) {
		unzipOptions.filePathMatcher = filePathMatcher
	}
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
	filePathMatcher func(string) bool,
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
	if filePathMatcher != nil && !filePathMatcher(fullPath) {
		return "", false, nil
	}
	return fullPath, true, nil
}

type untarOptions struct {
	maxFileSize         int64
	stripComponentCount uint32
	filePathMatcher     func(string) bool
}

func newUntarOptions() *untarOptions {
	return &untarOptions{}
}

type unzipOptions struct {
	stripComponentCount uint32
	filePathMatcher     func(string) bool
}

func newUnzipOptions() *unzipOptions {
	return &unzipOptions{}
}
