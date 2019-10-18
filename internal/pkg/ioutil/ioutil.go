// Package ioutil provides extensions to the stdlib ioutil package.
package ioutil

import (
	"errors"
	"io"
)

var (
	// DiscardReader is an io.Reader in which all calls return 0 and io.EOF.
	DiscardReader io.Reader = discardReader{}

	// ErrShortWrite is the errur returned from WriteAll if the amount written is less
	// than the input data length.
	ErrShortWrite = errors.New("wrote less bytes than requested")
)

// NopWriteCloser returns an io.WriteCloser with a no-op Close method wrapping the provided io.Writer.
func NopWriteCloser(writer io.Writer) io.WriteCloser {
	return nopWriteCloser{Writer: writer}
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

type discardReader struct{}

func (discardReader) Read([]byte) (int, error) {
	return 0, io.EOF
}
