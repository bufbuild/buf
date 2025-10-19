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

package bufpolicyconfig

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// File is a file.
type File interface {
	// FileVersion returns the file version.
	FileVersion() bufconfig.FileVersion
	// ObjectData returns the underlying ObjectData.
	//
	// This is non-nil on Files if they were created from storage.ReadBuckets. It is nil
	// if the File was created via a New constructor or Read method.
	//
	// This ObjectData is used for digest calculations.
	ObjectData() bufconfig.ObjectData

	isFile()
}

// *** PRIVATE ***

func getFile[F File](
	ctx context.Context,
	bucket storage.ReadBucket,
	path string,
	readFileFunc func(
		data []byte,
		objectData bufconfig.ObjectData,
		allowJSON bool,
	) (F, error),
) (F, error) {
	var f F
	data, err := storage.ReadPath(ctx, bucket, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return f, &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
		}
		var f F
		return f, err
	}
	f, err = readFileFunc(data, bufconfig.NewObjectData(path, data), false)
	if err != nil {
		return f, newDecodeError(path, err)
	}
	if err := validateSupportedFileVersion(path, f.FileVersion()); err != nil {
		return f, newDecodeError(path, err)
	}
	return f, nil
}

func putFile[F File](
	ctx context.Context,
	bucket storage.WriteBucket,
	path string,
	f F,
	writeFileFunc func(
		writer io.Writer,
		f F,
	) error,
) (retErr error) {
	if err := validateSupportedFileVersion(path, f.FileVersion()); err != nil {
		// This is effectively a system error. We should be able to write with whatever file name we have.
		return syserror.Wrap(newEncodeError(path, err))
	}
	writeObjectCloser, err := bucket.Put(ctx, path, storage.PutWithAtomic())
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, writeObjectCloser.Close())
	}()
	return writeFileFunc(writeObjectCloser, f)
}

func readFile[F File](
	reader io.Reader,
	fileName string,
	readFileFunc func(
		data []byte,
		objectData bufconfig.ObjectData,
		allowJSON bool,
	) (F, error),
) (F, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		var f F
		return f, err
	}
	objectData := bufconfig.NewObjectData(fileName, data)
	f, err := readFileFunc(data, objectData, true)
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

func getUnmarshalStrict(allowJSON bool) func([]byte, any) error {
	if allowJSON {
		return encoding.UnmarshalJSONOrYAMLStrict
	}
	return encoding.UnmarshalYAMLStrict
}

func getUnmarshalNonStrict(allowJSON bool) func([]byte, any) error {
	if allowJSON {
		return encoding.UnmarshalJSONOrYAMLNonStrict
	}
	return encoding.UnmarshalYAMLNonStrict
}

func newDecodeError(fileName string, err error) error {
	if fileName == "" {
		fileName = "policy file"
	}
	// We intercept PathErrors in buffetch to deal with fixing of paths.
	// We return a cleaned, unnormalized path in the error for clarity with user's filesystem.
	return &fs.PathError{Op: "decode", Path: filepath.Clean(normalpath.Unnormalize(fileName)), Err: err}
}

func newEncodeError(fileName string, err error) error {
	if fileName == "" {
		fileName = "policy file"
	}
	// We intercept PathErrors in buffetch to deal with fixing of paths.
	// We return a cleaned, unnormalized path in the error for clarity with user's filesystem.
	return &fs.PathError{Op: "encode", Path: filepath.Clean(normalpath.Unnormalize(fileName)), Err: err}
}
