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

package bufimage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	descriptorv1 "buf.build/gen/go/bufbuild/protodescriptor/protocolbuffers/go/buf/descriptor/v1"
	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/protocompile/experimental/fdp"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/bufbuild/protocompile/experimental/incremental/queries"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/source"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func buildImageExperimental(
	ctx context.Context,
	logger *slog.Logger,
	moduleReadBucket bufmodule.ModuleReadBucket,
	excludeSourceCodeInfo bool,
	noParallelism bool,
) (Image, error) {
	defer xslog.DebugProfile(logger)()

	if !moduleReadBucket.ShouldBeSelfContained() {
		return nil, syserror.New("passed a ModuleReadBucket to BuildImage that was not expected to be self-contained")
	}
	targetFileInfos, err := bufmodule.GetTargetFileInfos(ctx, moduleReadBucket)
	if err != nil {
		return nil, err
	}
	moduleReadBucket = bufmodule.ModuleReadBucketWithOnlyProtoFiles(moduleReadBucket)
	fmt.Println("COMPILING THE FOLLOWING:")
	for _, targetFileInfos := range targetFileInfos {
		fmt.Println("\t" + targetFileInfos.LocalPath())
	}

	fmt.Println("BUILDING")
	imageFiles, err := buildImageFiles(ctx, moduleReadBucket, targetFileInfos)
	if err != nil {
		return nil, err
	}
	return newImage(imageFiles, false, nil /* lazily constructed */)
}

func buildImageFiles(
	ctx context.Context,
	moduleReadBucket bufmodule.ModuleReadBucket,
	targetFileInfos []bufmodule.FileInfo,
) ([]ImageFile, error) {
	fileSet := make(map[string]struct{}, len(targetFileInfos))
	for _, targetFileInfo := range targetFileInfos {
		filePath := targetFileInfo.LocalPath()
		fileSet[filePath] = struct{}{}
	}

	moduleOpener := newModuleReadBucketOpener(ctx, moduleReadBucket)
	wktOpener := newReadBucketOpener(ctx, datawkt.ReadBucket)

	opener := &source.Openers{moduleOpener, wktOpener}

	exec := incremental.New(
		incremental.WithParallelism(1),
	)

	session := new(ir.Session)
	fileQueries := make([]incremental.Query[*ir.File], len(targetFileInfos))
	for index, targetFileInfo := range targetFileInfos {
		filePath := targetFileInfo.LocalPath()
		fileQueries[index] = queries.IR{
			Opener:  opener,
			Session: session,
			Path:    filePath,
		}
	}

	fmt.Println("RUNNING")
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	results, report, err := incremental.Run(ctx, exec, fileQueries...)
	if err != nil {
		return nil, err
	}
	_ = report // TODO: Capture report details
	fmt.Println("DONE")

	if n, m := len(results), len(fileQueries); n != m {
		return nil, fmt.Errorf("mismatch queries %d to results %d", m, n)
	}

	irFiles := make([]*ir.File, len(targetFileInfos))
	for index, result := range results {
		if result.Value != nil {
			filePath := targetFileInfos[index].LocalPath()
			return nil, fmt.Errorf("file %q returned zero result", filePath)
		}
		irFiles[index] = result.Value
	}

	// Convert to FileDescriptorSet.
	bytes, err := fdp.DescriptorSetBytes(
		irFiles,
		// Always enable source code info to extract the image options.
		fdp.IncludeSourceCodeInfo(true),
	)
	if err != nil {
		return nil, err
	}
	fds := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(bytes, fds); err != nil {
		return nil, err
	}

	// TODO: loop through and add the annotations etc...
	out, _ := protojson.MarshalOptions{
		Multiline: true,
	}.Marshal(fds)
	fmt.Println("GOT FILE DESCRIPTOR PROTO:\n", string(out))

	imageFiles := make([]ImageFile, len(fds.File))
	for index, fileDescriptor := range fds.File {
		targetFileInfo, err := moduleReadBucket.StatFileInfo(ctx, fileDescriptor.GetName())
		if err != nil {
			return nil, err
		}
		module := targetFileInfo.Module()

		// All input target files are _not_ imports.
		_, isNotImport := fileSet[fileDescriptor.GetName()]
		isImport := !isNotImport

		var (
			isSyntaxUnspecified     bool
			unusedDependencyIndexes []int32
		)
		sourcCodeInfo := fileDescriptor.GetSourceCodeInfo()
		if proto.HasExtension(sourcCodeInfo, descriptorv1.E_BufSourceCodeInfoExtension) {
			fmt.Println("HAS EXTENSION!")
			sourceCodeInfoExtAny := proto.GetExtension(sourcCodeInfo, descriptorv1.E_BufSourceCodeInfoExtension)
			sourceCodeInfoExt, ok := sourceCodeInfoExtAny.(*descriptorv1.SourceCodeInfoExtension)
			if !ok {
				panic(fmt.Sprintf("unexpected source code info extention type %T", sourceCodeInfoExtAny))
			}
			fmt.Println("Got", sourceCodeInfoExt)

			isSyntaxUnspecified = sourceCodeInfoExt.GetIsSyntaxUnspecified()
			unusedDependencyIndexes = sourceCodeInfoExt.GetUnusedDependency()
		}

		imageFile, err := NewImageFile(
			fileDescriptor,
			module.FullName(),
			module.CommitID(),
			targetFileInfo.ExternalPath(),
			targetFileInfo.LocalPath(),
			isImport,
			isSyntaxUnspecified,
			unusedDependencyIndexes,
		)
		if err != nil {
			return nil, err
		}
		imageFiles[index] = imageFile
	}
	return imageFiles, err
}

type moduleReadBucketOpener struct {
	context context.Context
	bucket  bufmodule.ModuleReadBucket
}

func newModuleReadBucketOpener(ctx context.Context, bucket bufmodule.ModuleReadBucket) *moduleReadBucketOpener {
	return &moduleReadBucketOpener{
		context: ctx,
		bucket:  bucket,
	}
}

func (r *moduleReadBucketOpener) Open(path string) (*source.File, error) {
	fmt.Println("MODULE OPENING", path)
	file, err := r.bucket.GetFile(r.context, path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buf strings.Builder
	if _, err = io.Copy(&buf, file); err != nil {
		return nil, err
	}
	return source.NewFile(path, buf.String()), nil
}

type readBucketOpener struct {
	context context.Context
	bucket  storage.ReadBucket
}

func newReadBucketOpener(ctx context.Context, bucket storage.ReadBucket) *readBucketOpener {
	return &readBucketOpener{
		context: ctx,
		bucket:  bucket,
	}
}

func (r *readBucketOpener) Open(path string) (*source.File, error) {
	fmt.Println("STORAGE OPENING", path)
	file, err := r.bucket.Get(r.context, path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buf strings.Builder
	if _, err = io.Copy(&buf, file); err != nil {
		return nil, err
	}
	return source.NewFile(path, buf.String()), nil
}
