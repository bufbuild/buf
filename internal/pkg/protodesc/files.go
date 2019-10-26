package protodesc

import (
	"runtime"

	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/protodescpb"
)

const defaultChunkSizeThreshold = 8

func newFiles(fileDescriptors ...protodescpb.FileDescriptor) ([]File, error) {
	if len(fileDescriptors) == 0 {
		return nil, nil
	}

	chunkSize := len(fileDescriptors) / runtime.NumCPU()
	if defaultChunkSizeThreshold != 0 && chunkSize < defaultChunkSizeThreshold {
		files := make([]File, 0, len(fileDescriptors))
		for _, fileDescriptor := range fileDescriptors {
			file, err := NewFile(fileDescriptor)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
		}
		return files, nil
	}

	chunks := fileDescriptorsToChunks(fileDescriptors, chunkSize)
	resultC := make(chan *result, len(chunks))
	for _, fileDescriptorChunk := range chunks {
		fileDescriptorChunk := fileDescriptorChunk
		go func() {
			files := make([]File, 0, len(fileDescriptorChunk))
			for _, fileDescriptor := range fileDescriptorChunk {
				file, err := NewFile(fileDescriptor)
				if err != nil {
					resultC <- newResult(nil, err)
					return
				}
				files = append(files, file)
			}
			resultC <- newResult(files, nil)
		}()
	}
	files := make([]File, 0, len(fileDescriptors))
	var err error
	for i := 0; i < len(chunks); i++ {
		result := <-resultC
		files = append(files, result.Files...)
		err = errs.Append(err, result.Err)
	}
	if err != nil {
		return nil, err
	}
	return files, nil
}

func fileDescriptorsToChunks(s []protodescpb.FileDescriptor, chunkSize int) [][]protodescpb.FileDescriptor {
	var chunks [][]protodescpb.FileDescriptor
	if len(s) == 0 {
		return chunks
	}
	if chunkSize <= 0 {
		return [][]protodescpb.FileDescriptor{s}
	}
	c := make([]protodescpb.FileDescriptor, len(s))
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
