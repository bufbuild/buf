package protodesc

import (
	"context"
	"runtime"

	protobufdescriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"go.uber.org/multierr"
)

const defaultChunkSizeThreshold = 8

func newFilesUnstable(ctx context.Context, fileDescriptorProtos ...*protobufdescriptor.FileDescriptorProto) ([]File, error) {
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

func fileDescriptorProtosToChunks(s []*protobufdescriptor.FileDescriptorProto, chunkSize int) [][]*protobufdescriptor.FileDescriptorProto {
	var chunks [][]*protobufdescriptor.FileDescriptorProto
	if len(s) == 0 {
		return chunks
	}
	if chunkSize <= 0 {
		return [][]*protobufdescriptor.FileDescriptorProto{s}
	}
	c := make([]*protobufdescriptor.FileDescriptorProto, len(s))
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
