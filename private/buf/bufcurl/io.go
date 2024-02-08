// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufcurl

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ErrorHasFilename makes sure that the given error includes a reference to the
// given filename. If not, it wraps the given error and adds the filename. This
// is to make sure errors are useful -- an error related to file I/O is not very
// helpful if it doesn't indicate the name of the file.
func ErrorHasFilename(err error, filename string) error {
	if strings.Contains(err.Error(), filename) {
		return err
	}
	return fmt.Errorf("%s: %w", filename, err)
}

type readerWithClose struct {
	io.Reader
	io.Closer
}

// lineReader wraps a *bufio.Reader, making it easier to read a file one line
// at a time.
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
