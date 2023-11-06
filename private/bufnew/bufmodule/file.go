package bufmodule

import "io"

// File is a file within a Module.
type File interface {
	FileInfo
	io.ReadCloser

	isFile()
}

// *** PRIVATE ***

type file struct {
	FileInfo
	io.ReadCloser
}

func newFile(fileInfo FileInfo, readCloser io.ReadCloser) *file {
	return &file{
		FileInfo:   fileInfo,
		ReadCloser: readCloser,
	}
}

func (*file) isFile() {}
