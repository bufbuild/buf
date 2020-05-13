// Copyright 2020 Buf Technologies Inc.
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

package protosrc

import (
	"context"
	"runtime"

	"go.uber.org/multierr"
	"google.golang.org/protobuf/types/descriptorpb"
)

const defaultChunkSizeThreshold = 8

func newFilesUnstable(ctx context.Context, fileDescriptorProtos ...*descriptorpb.FileDescriptorProto) ([]File, error) {
	if len(fileDescriptorProtos) == 0 {
		return nil, nil
	}

	chunkSize := len(fileDescriptorProtos) / runtime.NumCPU()
	if defaultChunkSizeThreshold != 0 && chunkSize < defaultChunkSizeThreshold {
		files := make([]File, 0, len(fileDescriptorProtos))
		for _, fileDescriptorProto := range fileDescriptorProtos {
			file, err := NewFile(fileDescriptorProto)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
		}
		return files, nil
	}

	chunks := fileDescriptorProtosToChunks(fileDescriptorProtos, chunkSize)
	resultC := make(chan *result, len(chunks))
	for _, fileDescriptorProtoChunk := range chunks {
		fileDescriptorProtoChunk := fileDescriptorProtoChunk
		go func() {
			files := make([]File, 0, len(fileDescriptorProtoChunk))
			for _, fileDescriptorProto := range fileDescriptorProtoChunk {
				file, err := NewFile(fileDescriptorProto)
				if err != nil {
					resultC <- newResult(nil, err)
					return
				}
				files = append(files, file)
			}
			resultC <- newResult(files, nil)
		}()
	}
	files := make([]File, 0, len(fileDescriptorProtos))
	var err error
	for i := 0; i < len(chunks); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-resultC:
			files = append(files, result.Files...)
			err = multierr.Append(err, result.Err)
		}
	}
	if err != nil {
		return nil, err
	}
	return files, nil
}

func fileDescriptorProtosToChunks(s []*descriptorpb.FileDescriptorProto, chunkSize int) [][]*descriptorpb.FileDescriptorProto {
	var chunks [][]*descriptorpb.FileDescriptorProto
	if len(s) == 0 {
		return chunks
	}
	if chunkSize <= 0 {
		return [][]*descriptorpb.FileDescriptorProto{s}
	}
	c := make([]*descriptorpb.FileDescriptorProto, len(s))
	copy(c, s)
	// https://github.com/golang/go/wiki/SliceTricks#batching-with-minimal-allocation
	for chunkSize < len(c) {
		c, chunks = c[chunkSize:], append(chunks, c[0:chunkSize:chunkSize])
	}
	return append(chunks, c)
}

type result struct {
	Files []File
	Err   error
}

func newResult(files []File, err error) *result {
	return &result{
		Files: files,
		Err:   err,
	}
}
