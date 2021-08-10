// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufimagebuildtesting

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/internal/buf/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/internal/buftesting"
	"github.com/bufbuild/buf/internal/pkg/prototesting"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/zap"
	"golang.org/x/tools/txtar"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Fuzz is the entrypoint for the fuzzer.
// We use https://github.com/dvyukov/go-fuzz for fuzzing.
// Please follow the instructions
// in their README for help with running the fuzz targets.
func Fuzz(data []byte) int {
	ctx := context.Background()
	result, err := fuzz(ctx, data)
	if err != nil {
		// data was invalid in some way
		return -1
	}
	return result.panicOrN(ctx)
}

func fuzz(ctx context.Context, data []byte) (_ *fuzzResult, err error) {
	dirPath, err := ioutil.TempDir("", "buffuzz")
	if err != nil {
		return nil, err
	}
	defer func() {
		e := os.RemoveAll(dirPath)
		if err == nil {
			err = e
		}
	}()
	err = untxtar(data, dirPath)
	if err != nil {
		return nil, err
	}

	filePaths, err := buftesting.GetProtocFilePathsErr(ctx, dirPath, 0)
	if err != nil {
		return nil, err
	}

	actualProtocFileDescriptorSet, protocErr := prototesting.GetProtocFileDescriptorSet(
		ctx,
		[]string{dirPath},
		filePaths,
		false,
		false,
	)

	image, bufAnnotations, bufErr := fuzzBuild(ctx, dirPath)
	return &fuzzResult{
		bufAnnotations:                bufAnnotations,
		bufErr:                        bufErr,
		protocErr:                     protocErr,
		actualProtocFileDescriptorSet: actualProtocFileDescriptorSet,
		image:                         image,
	}, nil
}

// fuzzBuild does a builder.Build for a fuzz test.
func fuzzBuild(ctx context.Context, dirPath string) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	moduleFileSet, err := fuzzGetModuleFileSet(ctx, dirPath)
	if err != nil {
		return nil, nil, err
	}
	builder := bufimagebuild.NewBuilder(zap.NewNop())
	opt := bufimagebuild.WithExcludeSourceCodeInfo()
	image, annotations, err := builder.Build(ctx, moduleFileSet, opt)
	return image, annotations, err
}

// fuzzGetModuleFileSet gets the bufmodule.ModuleFileSet for a fuzz test.
func fuzzGetModuleFileSet(ctx context.Context, dirPath string) (bufmodule.ModuleFileSet, error) {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	config, err := bufmodulebuild.NewConfigV1(bufmodulebuild.ExternalConfigV1{})
	if err != nil {
		return nil, err
	}
	module, err := bufmodulebuild.NewModuleBucketBuilder(zap.NewNop()).BuildForBucket(
		ctx,
		readWriteBucket,
		config,
	)
	if err != nil {
		return nil, err
	}
	return bufmodulebuild.NewModuleFileSetBuilder(
		zap.NewNop(),
		bufmodule.NewNopModuleReader(),
	).Build(
		ctx,
		module,
	)
}

// untxtar extracts txtar data to destDir.
func untxtar(data []byte, destDir string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	archive := txtar.Parse(data)
	if len(archive.Files) == 0 {
		return fmt.Errorf("txtar contains no files")
	}
	for _, file := range archive.Files {
		dir := filepath.Dir(file.Name)
		if dir != "" {
			err = os.MkdirAll(filepath.Join(destDir, dir), 0700)
			if err != nil {
				return err
			}
		}
		err = ioutil.WriteFile(
			filepath.Join(destDir, file.Name),
			file.Data,
			0600,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

type fuzzResult struct {
	bufAnnotations                []bufanalysis.FileAnnotation
	bufErr                        error
	protocErr                     error
	actualProtocFileDescriptorSet *descriptorpb.FileDescriptorSet
	image                         bufimage.Image
}

// panicOrN panics if there is an error or returns the appropriate value for Fuzz to return.
func (f *fuzzResult) panicOrN(ctx context.Context) int {
	err := f.error(ctx)
	if err != nil {
		panic(err.Error())
	}
	if f.protocErr == nil {
		return 1
	}
	return 0
}

// error returns an error that should cause Fuzz to panic.
func (f *fuzzResult) error(ctx context.Context) error {
	if f == nil {
		return nil
	}
	if f.protocErr != nil {
		if f.bufErr == nil && len(f.bufAnnotations) == 0 {
			return fmt.Errorf("protoc has error but buf does not: %v", f.protocErr)
		}
		return nil
	}
	if f.bufErr != nil {
		return fmt.Errorf("buf has error but protoc does not: %v", f.bufErr)
	}
	if len(f.bufAnnotations) > 0 {
		return fmt.Errorf("buf has file annotations but protoc has no error: %v", f.bufAnnotations)
	}
	image := bufimage.ImageWithoutImports(f.image)
	fileDescriptorSet := bufimage.ImageToFileDescriptorSet(image)

	diff, err := prototesting.DiffFileDescriptorSetsJSON(
		ctx,
		fileDescriptorSet,
		f.actualProtocFileDescriptorSet,
		"buf",
		"protoc",
	)
	if err != nil {
		return fmt.Errorf("error diffing results: %v", err)
	}
	if strings.TrimSpace(diff) != "" {
		return fmt.Errorf("protoc and buf have different results: %v", diff)
	}
	return nil
}
