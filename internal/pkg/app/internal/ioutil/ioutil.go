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

// Package ioutil provides io utilities.
package ioutil

import (
	"io"
)

var (
	// DiscardReader is an io.Reader in which all calls return 0 and io.EOF.
	DiscardReader io.Reader = discardReader{}
)

// NopWriteCloser returns an io.WriteCloser with a no-op Close method wrapping the provided io.Writer.
func NopWriteCloser(writer io.Writer) io.WriteCloser {
	return nopWriteCloser{Writer: writer}
}

type discardReader struct{}

func (discardReader) Read([]byte) (int, error) {
	return 0, io.EOF
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}
