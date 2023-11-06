package bufmodule

import "io"

// File is a file within a Module.
type File interface {
	FileInfo
	io.ReadCloser

	isFile()
}
