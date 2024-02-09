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

package bufconfig

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/multierr"
)

// File is the common interface shared by all config files.
type File interface {
	FileInfo

	// ObjectData returns the underlying ObjectData.
	//
	// This is non-nil on Files if they were created from storage.ReadBuckets. It is nil
	// if the File was created via a New constructor or Read method.
	//
	// This ObjectData is used for digest calculations.
	ObjectData() ObjectData

	isFile()
}

// *** PRIVATE ***

func getFileForPrefix[F File](
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	fileNames []string,
	fileNameToSupportedFileVersions map[string]map[FileVersion]struct{},
	readFileFunc func(
		data []byte,
		objectData ObjectData,
		allowJSON bool,
	) (F, error),
) (F, error) {
	for _, fileName := range fileNames {
		path := normalpath.Join(prefix, fileName)
		data, err := storage.ReadPath(ctx, bucket, path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			var f F
			return f, err
		}
		f, err := readFileFunc(data, newObjectData(fileName, data), false)
		if err != nil {
			return f, newDecodeError(path, err)
		}
		if err := validateSupportedFileVersion(fileName, f.FileVersion(), fileNameToSupportedFileVersions); err != nil {
			return f, newDecodeError(path, err)
		}
		return f, nil
	}
	var f F
	return f, &fs.PathError{Op: "read", Path: normalpath.Join(prefix, fileNames[0]), Err: fs.ErrNotExist}
}

func getFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	fileNames []string,
	fileNameToSupportedFileVersions map[string]map[FileVersion]struct{},
	fileVersionRequired bool,
	suggestedFileVersion FileVersion,
	defaultFileVersion FileVersion,
) (FileVersion, error) {
	for _, fileName := range fileNames {
		path := normalpath.Join(prefix, fileName)
		data, err := storage.ReadPath(ctx, bucket, path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return 0, err
		}
		fileVersion, err := getFileVersionForData(data, false, fileVersionRequired, fileNameToSupportedFileVersions, suggestedFileVersion, defaultFileVersion)
		if err != nil {
			return 0, newDecodeError(path, err)
		}
		if err := validateSupportedFileVersion(fileName, fileVersion, fileNameToSupportedFileVersions); err != nil {
			return 0, newDecodeError(path, err)
		}
		return fileVersion, nil
	}
	return 0, &fs.PathError{Op: "read", Path: normalpath.Join(prefix, fileNames[0]), Err: fs.ErrNotExist}
}

func putFileForPrefix[F File](
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	f F,
	fileName string,
	fileNameToSupportedFileVersions map[string]map[FileVersion]struct{},
	writeFileFunc func(
		writer io.Writer,
		f F,
	) error,
) (retErr error) {
	if err := validateSupportedFileVersion(fileName, f.FileVersion(), fileNameToSupportedFileVersions); err != nil {
		// This is effectively a system error. We should be able to write with whatever file name we have.
		return syserror.Wrap(newEncodeError(fileName, err))
	}
	path := normalpath.Join(prefix, fileName)
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
	fileName string,
	readFileFunc func(
		data []byte,
		objectData ObjectData,
		allowJSON bool,
	) (F, error),
) (F, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		var f F
		return f, err
	}
	f, err := readFileFunc(data, nil, true)
	if err != nil {
		return f, newDecodeError(fileName, err)
	}
	return f, nil
}

func writeFile[F File](
	writer io.Writer,
	f F,
	writeFileFunc func(
		writer io.Writer,
		f F,
	) error,
) error {
	if err := writeFileFunc(writer, f); err != nil {
		var fileName string
		if objectData := f.ObjectData(); objectData != nil {
			fileName = objectData.Name()
		}
		return newDecodeError(fileName, err)
	}
	return nil
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

func newDecodeError(fileName string, err error) error {
	if fileName == "" {
		fileName = "config file"
	}
	// We intercept PathErrors in buffetch to deal with fixing of paths.
	// We return a cleaned, unnormalized path in the error for clarity with user's filesystem.
	return &fs.PathError{Op: "decode", Path: filepath.Clean(normalpath.Unnormalize(fileName)), Err: err}
}

func newEncodeError(fileName string, err error) error {
	if fileName == "" {
		fileName = "config file"
	}
	// We intercept PathErrors in buffetch to deal with fixing of paths.
	// We return a cleaned, unnormalized path in the error for clarity with user's filesystem.
	return &fs.PathError{Op: "encode", Path: filepath.Clean(normalpath.Unnormalize(fileName)), Err: err}
}
