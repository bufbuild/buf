package app

import (
	"io"

	internalioutil "github.com/bufbuild/buf/internal/pkg/app/internal/ioutil"
)

type stdinContainer struct {
	reader io.Reader
}

func newStdinContainer(reader io.Reader) *stdinContainer {
	if reader == nil {
		reader = internalioutil.DiscardReader
	}
	return &stdinContainer{
		reader: reader,
	}
}

func (s *stdinContainer) Stdin() io.Reader {
	return s.reader
}
