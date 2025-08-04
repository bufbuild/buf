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

package bufprotosource

import (
	"context"
	"sync"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/thread"
	"google.golang.org/protobuf/reflect/protodesc"
)

const defaultChunkSizeThreshold = 8

func newFiles[F InputFile](
	ctx context.Context,
	inputFiles []F,
	resolver protodesc.Resolver,
) ([]File, error) {
	indexedInputFiles := xslices.ToIndexed(inputFiles)
	if len(indexedInputFiles) == 0 {
		return nil, nil
	}

	// Why were we chunking this? We could just send each individual call to thread.Parallelize
	// and let thread.Parallelize deal with what to do.

	chunkSize := len(indexedInputFiles) / thread.Parallelism()
	if chunkSize < defaultChunkSizeThreshold {
		files := make([]File, 0, len(indexedInputFiles))
		for _, indexedInputFile := range indexedInputFiles {
			file, err := newFile(indexedInputFile.Value, resolver)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
		}
		return files, nil
	}
	chunks := xslices.ToChunks(indexedInputFiles, chunkSize)
	indexedFiles := make([]xslices.Indexed[File], 0, len(indexedInputFiles))
	jobs := make([]func(context.Context) error, len(chunks))
	var lock sync.Mutex
	for i, indexedInputFileChunk := range chunks {
		jobs[i] = func(ctx context.Context) error {
			iIndexedFiles := make([]xslices.Indexed[File], 0, len(indexedInputFileChunk))
			for _, indexedInputFile := range indexedInputFileChunk {
				file, err := newFile(indexedInputFile.Value, resolver)
				if err != nil {
					return err
				}
				iIndexedFiles = append(iIndexedFiles, xslices.Indexed[File]{Value: file, Index: indexedInputFile.Index})
			}
			lock.Lock()
			indexedFiles = append(indexedFiles, iIndexedFiles...)
			lock.Unlock()
			return nil
		}
	}
	if err := thread.Parallelize(ctx, jobs); err != nil {
		return nil, err
	}
	return xslices.IndexedToSortedValues(indexedFiles), nil
}
