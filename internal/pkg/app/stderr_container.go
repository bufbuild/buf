package app

import (
	"io"
	"io/ioutil"
)

type stderrContainer struct {
	writer io.Writer
}

func newStderrContainer(writer io.Writer) *stderrContainer {
	if writer == nil {
		writer = ioutil.Discard
	}
	return &stderrContainer{
		writer: writer,
	}
}

func (s *stderrContainer) Stderr() io.Writer {
	return s.writer
}
