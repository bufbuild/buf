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

package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"go.uber.org/zap"
)

type protoFileWriter struct {
	logger *zap.Logger
}

func newProtoFileWriter(
	logger *zap.Logger,
) *protoFileWriter {
	return &protoFileWriter{
		logger: logger,
	}
}

func (w *protoFileWriter) PutProtoFile(
	ctx context.Context,
	container app.EnvStdoutContainer,
	protoFileRef ProtoFileRef,
) (io.WriteCloser, error) {
	switch fileScheme := protoFileRef.FileScheme(); fileScheme {
	case FileSchemeHTTP:
		return nil, fmt.Errorf("http not supported for writes: %v", protoFileRef.Path())
	case FileSchemeHTTPS:
		return nil, fmt.Errorf("https not supported for writes: %v", protoFileRef.Path())
	case FileSchemeLocal:
		filePath := protoFileRef.Path()
		if filePath == "" {
			return nil, fmt.Errorf("empty path for ProtoFileRef: %v", protoFileRef)
		}
		if dirPath := normalpath.Dir(filePath); dirPath != "." {
			if err := createDirIfNotExists(normalpath.Unnormalize(dirPath)); err != nil {
				return nil, err
			}
		}
		// This was in the old formatter, just keeping as-is for now instead of using os.Create.
		return os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	case FileSchemeStdio, FileSchemeStdout:
		return ioext.NopWriteCloser(container.Stdout()), nil
	case FileSchemeStdin:
		return nil, errors.New("cannot write to stdin")
	case FileSchemeNull:
		return ioext.DiscardWriteCloser, nil
	default:
		return nil, fmt.Errorf("unknown FileScheme: %v", fileScheme)
	}
}

func createDirIfNotExists(dirPath string) error {
	// OK to use os.Stat instead of os.LStat here as this is CLI-only
	if _, err := os.Stat(dirPath); err != nil {
		// We don't need to check fileInfo.IsDir() because it's
		// already handled by the storageosProvider.
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return err
			}
			// We could os.RemoveAll if the overall command exits without error, but we're
			// not going to, just to be safe.
		}
	}
	return nil
}
