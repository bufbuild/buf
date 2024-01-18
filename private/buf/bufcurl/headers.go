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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/multierr"
)

// headersBlockList contains disallowed headers. These are headers that are part
// of the Connect or gRPC protocol and set by the protocol implementations, so
// should not be set otherwise. It also includes "transfer-encoding", which is
// not part of either protocol, but is unsafe for users to set as it handled
// by the user agent.
//
// In addition to these headers, header names that start with "Connect-" and
// "Grpc-" are also reserved for use by protocol implementations.
var headerBlockList = map[string]struct{}{
	"accept":            {},
	"accept-encoding":   {},
	"content-type":      {},
	"content-encoding":  {},
	"te":                {},
	"transfer-encoding": {},
}

// GetAuthority determines the authority for a request with the given URL and
// request headers. If headers include a "Host" header, that is used. (If the
// request contains more than one, that is usually not valid or acceptable to
// servers, but this function will look at only the first.) If there is no
// such header, the authority is the host portion of the URL (both the domain
// name/IP address and port).
func GetAuthority(url *url.URL, headers http.Header) string {
	header := headers.Get("host")
	if header != "" {
		return header
	}
	return url.Host
}

// LoadHeaders computes the set of request headers from the given flag values,
// loading from file(s) if so instructed. A header flag is usually in the form
// "name: value", but it may start with "@" to indicate a filename from which
// headers are loaded. It may also be "*", to indicate that the given others
// are included in full.
//
// If the filename following an "@" header flag is "-", it means to read from
// stdin.
//
// The given dataFile is the name of a file from which request data is read. If
// a "@" header flag indicates to read from the same file, then the headers must
// be at the start of the file, following by a blank line, followed by the
// actual request body. In such a case, the returned ReadCloser will be non-nil
// and correspond to that point in the file (after headers and blank line), so
// the request body can be read from it.
func LoadHeaders(headerFlags []string, dataFile string, others http.Header) (http.Header, io.ReadCloser, error) {
	var dataReader io.ReadCloser
	headers := http.Header{}
	for _, headerFlag := range headerFlags {
		switch {
		case strings.HasPrefix(headerFlag, "@"):
			headerFile := strings.TrimPrefix(headerFlag, "@")
			if headerFile != "-" {
				if absFile, err := filepath.Abs(headerFile); err == nil {
					headerFile = absFile
				}
			}
			isAlsoDataFile := headerFile == dataFile
			reader, err := readHeadersFile(headerFile, isAlsoDataFile, headers)
			if err != nil {
				return nil, nil, err
			}
			if isAlsoDataFile {
				dataReader = reader
			}
		case headerFlag == "*":
			for k, v := range others {
				headers[k] = append(headers[k], v...)
			}
		default:
			addHeader(headerFlag, headers)
		}
	}
	// make sure there are no disallowed headers used
	for key := range headers {
		lowerKey := strings.ToLower(key)
		if _, ok := headerBlockList[lowerKey]; ok || strings.HasPrefix(lowerKey, "grpc-") || strings.HasPrefix(lowerKey, "connect-") {
			return nil, nil, fmt.Errorf("invalid header: %q is reserved and may not be used", key)
		}
	}
	return headers, dataReader, nil
}

func readHeadersFile(headerFile string, stopAtBlankLine bool, headers http.Header) (reader io.ReadCloser, err error) {
	var f *os.File
	if headerFile == "-" {
		f = os.Stdin
	} else {
		f, err = os.Open(headerFile)
		if err != nil {
			return nil, ErrorHasFilename(err, headerFile)
		}
	}
	defer func() {
		if f != nil {
			closeErr := f.Close()
			if closeErr != nil {
				err = multierr.Append(err, ErrorHasFilename(closeErr, headerFile))
			}
		}
	}()
	in := &lineReader{r: bufio.NewReader(f)}
	var lineNo int
	for {
		line, err := in.ReadLine()
		if err == io.EOF {
			if stopAtBlankLine {
				// never hit a blank line
				return nil, ErrorHasFilename(io.ErrUnexpectedEOF, headerFile)
			}
			return nil, nil
		} else if err != nil {
			return nil, ErrorHasFilename(err, headerFile)
		}
		if line == "" && stopAtBlankLine {
			closer := f
			// we don't want close f on return in above defer function, so we clear it
			f = nil
			return &readerWithClose{Reader: in, Closer: closer}, nil
		}
		line = strings.TrimSpace(line)
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			// ignore blank lines and shell-style comments
			continue
		}
		if !addHeader(line, headers) {
			return nil, fmt.Errorf("%s:%d: malformed header: %q", headerFile, lineNo, line)
		}
		lineNo++
	}
}

func addHeader(header string, dest http.Header) bool {
	parts := strings.SplitN(header, ":", 2)
	headerName := parts[0]
	hasValue := len(parts) > 1
	var headerVal string
	if hasValue {
		headerVal = parts[1]
	}
	dest.Add(strings.TrimSpace(headerName), strings.TrimSpace(headerVal))
	return hasValue
}
