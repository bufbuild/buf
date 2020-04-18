package app

import (
	"io"
	"io/ioutil"
)

type stdoutContainer struct {
	writer io.Writer
}

func newStdoutContainer(writer io.Writer) *stdoutContainer {
	if writer == nil {
		writer = ioutil.Discard
	}
	return &stdoutContainer{
		writer: writer,
	}
}

func (s *stdoutContainer) Stdout() io.Writer {
	return s.writer
}
