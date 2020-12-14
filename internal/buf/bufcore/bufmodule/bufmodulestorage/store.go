// Copyright 2020 Buf Technologies, Inc.
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

package bufmodulestorage

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	moduleFileName = "module.bin.zst"
	formatVersion  = "v1"
)

type store struct {
	logger          *zap.Logger
	readWriteBucket storage.ReadWriteBucket
}

func newStore(logger *zap.Logger, readWriteBucket storage.ReadWriteBucket) *store {
	// Add version prefix
	readWriteBucket = storage.MapReadWriteBucket(readWriteBucket, storage.MapOnPrefix(formatVersion))
	return &store{
		logger:          logger.Named("bufmodulestorage"),
		readWriteBucket: readWriteBucket,
	}
}

func (s *store) Get(ctx context.Context, key Key) (_ bufmodule.Module, retErr error) {
	if err := normalpath.ValidatePathComponents(key...); err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}
	modulePath := normalpath.Join(append(key, moduleFileName)...)
	s.logger.Debug("get", zap.String("path", modulePath))
	readObjectCloser, err := s.readWriteBucket.Get(ctx, modulePath)
	if err != nil {
		// This correctly returns an error that fufills storage.ErrNotExist per the documentation
		// if the module is not in the reader.
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readObjectCloser.Close())
	}()
	zstdDecoder, err := zstd.NewReader(readObjectCloser)
	if err != nil {
		return nil, err
	}
	defer zstdDecoder.Close()
	data, err := ioutil.ReadAll(zstdDecoder)
	if err != nil {
		return nil, err
	}
	protoModule := &modulev1.Module{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, protoModule); err != nil {
		return nil, err
	}
	return bufmodule.NewModuleForProto(ctx, protoModule)
}

func (s *store) Put(ctx context.Context, key Key, module bufmodule.Module) (retErr error) {
	if err := normalpath.ValidatePathComponents(key...); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}
	modulePath := normalpath.Join(append(key, moduleFileName)...)
	s.logger.Debug("put", zap.String("path", modulePath))
	// Check if there is already a module at the path
	if _, err := s.readWriteBucket.Stat(ctx, modulePath); !storage.IsNotExist(err) {
		return fmt.Errorf("module already exists at path %q", modulePath)
	}
	protoModule, err := bufmodule.ModuleToProtoModule(ctx, module)
	if err != nil {
		return err
	}
	data, err := protoencoding.NewWireMarshaler().Marshal(protoModule)
	if err != nil {
		return err
	}
	writeObjectCloser, err := s.readWriteBucket.Put(ctx, modulePath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeObjectCloser.Close())
	}()
	zstdEncoder, err := zstd.NewWriter(writeObjectCloser)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, zstdEncoder.Close())
	}()
	if _, err := zstdEncoder.Write(data); err != nil {
		return err
	}
	return nil
}

func (s *store) Delete(ctx context.Context, key Key) error {
	if err := normalpath.ValidatePathComponents(key...); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}
	modulePath := normalpath.Join(append(key, moduleFileName)...)
	s.logger.Debug("delete", zap.String("path", modulePath))
	return s.readWriteBucket.Delete(ctx, modulePath)
}

func (s *store) AllKeys(ctx context.Context) ([]Key, error) {
	paths, err := storage.AllPaths(ctx, s.readWriteBucket, "")
	if err != nil {
		return nil, err
	}
	var keys []Key
	for _, path := range paths {
		if normalpath.Base(path) != moduleFileName {
			return nil, fmt.Errorf("unexpected path in bucket: %q", path)
		}
		// Trim module file name
		dir := normalpath.Dir(path)
		components := normalpath.Components(dir)
		keys = append(keys, Key(components))
	}
	return keys, nil
}
