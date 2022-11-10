// Copyright 2020-2022 Buf Technologies, Inc.
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

package curl

import (
	"bufio"
	"io"
	"strings"
)

func errorHasFilename(err error, filename string) error {
	if strings.Contains(err.Error(), filename) {
		return err
	}
	return &errorWithFilename{err: err, filename: filename}
}

type errorWithFilename struct {
	err      error
	filename string
}

func (e *errorWithFilename) Error() string {
	return e.filename + ": " + e.err.Error()
}

func (e *errorWithFilename) Unwrap() error {
	return e.err
}

type readerWithClose struct {
	io.Reader
	io.Closer
}

type lineReader struct {
	r   *bufio.Reader
	err error
}

func (r *lineReader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *lineReader) ReadLine() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	str, err := r.r.ReadString('\n')
	// Instead of returning data AND error, like bufio.Reader.ReadString,
	// only return one or the other since that is easier for the caller.
	if err != nil {
		if str != "" {
			r.err = err // save for next call
			return str, nil
		}
		return "", err
	}
	// If bufio.Reader.ReadString returns nil err, then the string ends
	// with the delimiter. Remove it.
	str = strings.TrimSuffix(str, "\n")
	return str, nil
}
