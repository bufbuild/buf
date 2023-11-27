// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufconfig

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/multierr"
)

// File is the common interface shared by all config files.
type File interface {
	// FileVersion returns the version of the file.
	FileVersion() FileVersion

	isFile()
}

// *** PRIVATE ***

func getFileForPrefix[F File](
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	fileNames []*fileName,
	readFileFunc func(
		reader io.Reader,
		allowJSON bool,
	) (F, error),
) (F, error) {
	for _, fileName := range fileNames {
		path := normalpath.Join(prefix, fileName.Name())
		readObjectCloser, err := bucket.Get(ctx, path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			var f F
			return f, err
		}
		f, err := readFileFunc(readObjectCloser, false)
		if err != nil {
			return f, multierr.Append(newDecodeError(path, err), readObjectCloser.Close())
		}
		if err := fileName.CheckSupportedFile(f); err != nil {
			return f, multierr.Append(newDecodeError(path, err), readObjectCloser.Close())
		}
		return f, readObjectCloser.Close()
	}
	var f F
	return f, &fs.PathError{Op: "read", Path: normalpath.Join(prefix, fileNames[0].Name()), Err: fs.ErrNotExist}
}

func getFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	fileNames []*fileName,
) (FileVersion, error) {
	for _, fileName := range fileNames {
		path := normalpath.Join(prefix, fileName.Name())
		data, err := storage.ReadPath(ctx, bucket, path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return 0, err
		}
		fileVersion, err := getFileVersionForData(data, false)
		if err != nil {
			return 0, newDecodeError(path, err)
		}
		if err := fileName.CheckSupportedFileVersion(fileVersion); err != nil {
			return 0, newDecodeError(path, err)
		}
		return fileVersion, nil
	}
	return 0, &fs.PathError{Op: "read", Path: normalpath.Join(prefix, fileNames[0].Name()), Err: fs.ErrNotExist}
}

func putFileForPrefix[F File](
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	f F,
	fileName *fileName,
	writeFileFunc func(
		writer io.Writer,
		f F,
	) error,
) (retErr error) {
	if err := fileName.CheckSupportedFile(f); err != nil {
		// This is effectively a system error. We should be able to write with whatever file name we have.
		return syserror.Wrap(newEncodeError(fileName.Name(), err))
	}
	path := normalpath.Join(prefix, fileName.Name())
	writeObjectCloser, err := bucket.Put(ctx, path, storage.PutWithAtomic())
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeObjectCloser.Close())
	}()
	return writeFileFunc(writeObjectCloser, f)
}

func readFile[F File](
	reader io.Reader,
	fileIdentifier string,
	readFileFunc func(
		reader io.Reader,
		allowJSON bool,
	) (F, error),
) (F, error) {
	f, err := readFileFunc(reader, true)
	if err != nil {
		return f, newDecodeError(fileIdentifier, err)
	}
	if err := checkV2SupportedYet(f.FileVersion()); err != nil {
		return f, newDecodeError(fileIdentifier, err)
	}
	return f, nil
}

func writeFile[F File](
	writer io.Writer,
	fileIdentifier string,
	f F,
	writeFileFunc func(
		writer io.Writer,
		f F,
	) error,
) error {
	if err := checkV2SupportedYet(f.FileVersion()); err != nil {
		return newEncodeError(fileIdentifier, err)
	}
	return writeFileFunc(writer, f)
}

func getFileVersionForData(
	data []byte,
	allowJSON bool,
) (FileVersion, error) {
	var externalFileVersion externalFileVersion
	if err := getUnmarshalNonStrict(allowJSON)(data, &externalFileVersion); err != nil {
		return 0, err
	}
	return parseFileVersion(externalFileVersion.Version)
}

func getUnmarshalStrict(allowJSON bool) func([]byte, interface{}) error {
	if allowJSON {
		return encoding.UnmarshalJSONOrYAMLStrict
	}
	return encoding.UnmarshalYAMLStrict
}

func getUnmarshalNonStrict(allowJSON bool) func([]byte, interface{}) error {
	if allowJSON {
		return encoding.UnmarshalJSONOrYAMLNonStrict
	}
	return encoding.UnmarshalYAMLNonStrict
}

func newDecodeError(fileIdentifier string, err error) error {
	return fmt.Errorf("failed to decode %s: %w", fileIdentifier, err)
}

func newEncodeError(fileIdentifier string, err error) error {
	return fmt.Errorf("failed to encode %s: %w", fileIdentifier, err)
}
