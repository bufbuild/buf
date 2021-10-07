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
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
	"github.com/bufbuild/buf/private/bufpkg/buftesting"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/prototesting"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tmp"
	"go.uber.org/multierr"
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
	runner := command.NewRunner()
	result, err := fuzz(ctx, runner, data)
	if err != nil {
		// data was invalid in some way
		return -1
	}
	return result.panicOrN(ctx)
}

func fuzz(ctx context.Context, runner command.Runner, data []byte) (_ *fuzzResult, retErr error) {
	dir, err := tmp.NewDir()
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, dir.Close())
	}()
	if err := untxtar(data, dir.AbsPath()); err != nil {
		return nil, err
	}

	filePaths, err := buftesting.GetProtocFilePathsErr(ctx, dir.AbsPath(), 0)
	if err != nil {
		return nil, err
	}

	actualProtocFileDescriptorSet, protocErr := prototesting.GetProtocFileDescriptorSet(
		ctx,
		runner,
		[]string{dir.AbsPath()},
		filePaths,
		false,
		false,
	)

	image, bufAnnotations, bufErr := fuzzBuild(ctx, dir.AbsPath())
	return newFuzzResult(
		runner,
		bufAnnotations,
		bufErr,
		protocErr,
		actualProtocFileDescriptorSet,
		image,
	), nil
}

// fuzzBuild does a builder.Build for a fuzz test.
func fuzzBuild(ctx context.Context, dirPath string) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	moduleFileSet, err := fuzzGetModuleFileSet(ctx, dirPath)
	if err != nil {
		return nil, nil, err
	}
	builder := bufimagebuild.NewBuilder(zap.NewNop())
	opt := bufimagebuild.WithExcludeSourceCodeInfo()
	return builder.Build(ctx, moduleFileSet, opt)
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
	config, err := bufmoduleconfig.NewConfigV1(bufmoduleconfig.ExternalConfigV1{})
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

// txtarParse is a wrapper around txtar.Parse that will turn panics into errors.
// This is necessary because of an issue where txtar.Parse can panic on invalid data. Because data is generated by the
// fuzzer, it will occasionally generate data that causes this panic.
// See https://github.com/golang/go/issues/47193
func txtarParse(data []byte) (_ *txtar.Archive, retErr error) {
	defer func() {
		if p := recover(); p != nil {
			retErr = fmt.Errorf("panic from txtar.Parse: %v", p)
		}
	}()
	return txtar.Parse(data), nil
}

// untxtar extracts txtar data to destDirPath.
func untxtar(data []byte, destDirPath string) error {
	archive, err := txtarParse(data)
	if err != nil {
		return err
	}
	if len(archive.Files) == 0 {
		return fmt.Errorf("txtar contains no files")
	}
	for _, file := range archive.Files {
		dirPath := filepath.Dir(file.Name)
		if dirPath != "" {
			if err := os.MkdirAll(filepath.Join(destDirPath, dirPath), 0700); err != nil {
				return err
			}
		}
		if err := os.WriteFile(
			filepath.Join(destDirPath, file.Name),
			file.Data,
			0600,
		); err != nil {
			return err
		}
	}
	return nil
}

type fuzzResult struct {
	runner                        command.Runner
	bufAnnotations                []bufanalysis.FileAnnotation
	bufErr                        error
	protocErr                     error
	actualProtocFileDescriptorSet *descriptorpb.FileDescriptorSet
	image                         bufimage.Image
}

func newFuzzResult(
	runner command.Runner,
	bufAnnotations []bufanalysis.FileAnnotation,
	bufErr error,
	protocErr error,
	actualProtocFileDescriptorSet *descriptorpb.FileDescriptorSet,
	image bufimage.Image,
) *fuzzResult {
	return &fuzzResult{
		runner:                        runner,
		bufAnnotations:                bufAnnotations,
		bufErr:                        bufErr,
		protocErr:                     protocErr,
		actualProtocFileDescriptorSet: actualProtocFileDescriptorSet,
		image:                         image,
	}
}

// panicOrN panics if there is an error or returns the appropriate value for Fuzz to return.
func (f *fuzzResult) panicOrN(ctx context.Context) int {
	if err := f.error(ctx); err != nil {
		panic(err.Error())
	}
	// This will return 1 for valid protobufs and 0 for invalid in order to encourage the fuzzer to generate more
	// realistic looking data.
	if f.protocErr == nil {
		return 1
	}
	return 0
}

// error returns an error that should cause Fuzz to panic.
func (f *fuzzResult) error(ctx context.Context) error {
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
		f.runner,
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
