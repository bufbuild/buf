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

package bufprotopluginexec

import (
	"bytes"
	"errors"
	"io"
)

type protocGenSwiftStderrWriteCloser struct {
	delegate io.Writer
	buffer   *bytes.Buffer
}

func newProtocGenSwiftStderrWriteCloser(delegate io.Writer) io.WriteCloser {
	return &protocGenSwiftStderrWriteCloser{
		delegate: delegate,
		buffer:   bytes.NewBuffer(nil),
	}
}

func (p *protocGenSwiftStderrWriteCloser) Write(data []byte) (int, error) {
	// If protoc-gen-swift, we want to capture all the stderr so we can process it.
	return p.buffer.Write(data)
}

func (p *protocGenSwiftStderrWriteCloser) Close() error {
	data := p.buffer.Bytes()
	if len(data) == 0 {
		return nil
	}
	newData := bytes.ReplaceAll(
		data,
		// If swift-protobuf changes their error message, this may not longer filter properly
		// but this is OK - this filtering should be treated as non-critical.
		// https://github.com/apple/swift-protobuf/blob/c3d060478fcf1f564be0a3876bde8c04247793ae/Sources/protoc-gen-swift/main.swift#L244
		//
		// Note that our heuristic as to whether this is protoc-gen-swift or not for isProtocGenSwift
		// is that the binary is named protoc-gen-swift, and protoc-gen-swift will print the binary name
		// before any message to stderr, so given our protoc-gen-swift heuristic, this is the
		// error message that will be printed.
		// https://github.com/apple/swift-protobuf/blob/c3d060478fcf1f564be0a3876bde8c04247793ae/Sources/protoc-gen-swift/FileIo.swift#L19
		//
		// Tested manually on Mac.
		// TODO: Test manually on Windows.
		[]byte("protoc-gen-swift: WARNING: unknown version of protoc, use 3.2.x or later to ensure JSON support is correct.\n"),
		nil,
	)
	if len(newData) == 0 {
		return nil
	}
	n, err := p.delegate.Write(newData)
	if err != nil {
		return err
	}
	if n != len(newData) {
		return errors.New("incomplete write")
	}
	return nil
}
