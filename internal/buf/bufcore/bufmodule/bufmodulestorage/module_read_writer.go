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
	"bytes"
	"context"
	"io/ioutil"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/multierr"
)

const (
	moduleFileName = "module.bin.zst"
)

type moduleReadWriter struct {
	// We want to make this a combined ReadWriteBucket as opposed to splitting
	// moduleReader and moduleWriter as we may want to auto-correct on GetModule
	// and delete modules that do not have equivalent digests in the future.
	readWriteBucket storage.ReadWriteBucket
}

func newModuleReadWriter(
	readWriteBucket storage.ReadWriteBucket,
) *moduleReadWriter {
	return &moduleReadWriter{
		readWriteBucket: readWriteBucket,
	}
}

func (m *moduleReadWriter) GetModule(
	ctx context.Context,
	moduleName bufmodule.ModuleName,
) (_ bufmodule.Module, retErr error) {
	if moduleName.Digest() == "" {
		return nil, bufmodule.NewNoDigestError(moduleName)
	}
	readObjectCloser, err := m.readWriteBucket.Get(
		ctx,
		getModuleFilePath(moduleName),
	)
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
	module, err := bufmodule.NewModuleForProto(ctx, protoModule)
	if err != nil {
		return nil, err
	}
	if err := bufmodule.ValidateModuleDigest(ctx, moduleName, module); err != nil {
		// We likely want to remove the module from the cache here.
		// This requires adding a Delete method to WriteBucket, which has various implications.
		// For example, what do we do about directories managed by storageos WriteBuckets
		// if we delete? We would theoretically want to delete them, but storageos buckets
		// are used in different ways, in some cases, we don't want to mess with the
		// underlying Bucket directory structure. If we don't delete though, we will have
		// "orphan" directories which could take up space over time.
		return nil, err
	}
	return module, nil
}

func (m *moduleReadWriter) PutModule(
	ctx context.Context,
	moduleName bufmodule.ModuleName,
	module bufmodule.Module,
) (_ bufmodule.ModuleName, retErr error) {
	resolvedModuleName, err := bufmodule.ResolvedModuleNameForModule(ctx, moduleName, module)
	if err != nil {
		return nil, err
	}
	protoModule, err := bufmodule.ModuleToProtoModule(ctx, module)
	if err != nil {
		return nil, err
	}
	data, err := protoencoding.NewWireMarshaler().Marshal(protoModule)
	if err != nil {
		return nil, err
	}
	// We need to know the size before writing to the bucket so we have to do this.
	// It would be a lot nicer if we could pipeline all of this as this uses
	// a lot of memory.
	buffer := bytes.NewBuffer(nil)
	zstdEncoder, err := zstd.NewWriter(buffer)
	if err != nil {
		return nil, err
	}
	if _, err := zstdEncoder.Write(data); err != nil {
		return nil, multierr.Append(err, zstdEncoder.Close())
	}
	// We have to make sure the encoder is closed before reading the buffer so
	// that we make sure it is flushed.
	if err := zstdEncoder.Close(); err != nil {
		return nil, err
	}
	compressedData := buffer.Bytes()
	// We might want to not just overwrite blindly here and instead return early
	// if the module is already stored, however this auto-corrects if something
	// was incorrectly stored and would not pass validation.
	writeObjectCloser, err := m.readWriteBucket.Put(
		ctx,
		getModuleFilePath(resolvedModuleName),
		uint32(len(compressedData)),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeObjectCloser.Close())
	}()
	if _, err := writeObjectCloser.Write(compressedData); err != nil {
		return nil, err
	}
	return resolvedModuleName, nil
}

// this assumes the ModuleName is resolved
func getModuleFilePath(moduleName bufmodule.ModuleName) string {
	// We should know that all of these names are valid file path components.
	// We probably already know this due to validation rules we apply to ModuleName
	// ie the characters used are restricted, and all of these are non-empty,
	// digest is the weird one though.
	return normalpath.Join(
		moduleName.Server(),
		moduleName.Owner(),
		moduleName.Repository(),
		moduleName.Version(),
		moduleName.Digest(),
		moduleFileName,
	)
}
