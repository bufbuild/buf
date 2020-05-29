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

package protoc

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/proto/protoencoding"
	"google.golang.org/protobuf/types/descriptorpb"
)

// GetFileDescriptorSet gets the validated FileDescriptorSet using
// protoc and the Well-Known Types on the current PATH.
//
// Only use for testing.
func GetFileDescriptorSet(
	ctx context.Context,
	roots []string,
	realFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) (_ *descriptorpb.FileDescriptorSet, retErr error) {
	protocBinPath, err := exec.LookPath("protoc")
	if err != nil {
		return nil, err
	}
	protocBinPath, err = filepath.Abs(protocBinPath)
	if err != nil {
		return nil, err
	}
	protocIncludePath, err := filepath.Abs(filepath.Join(filepath.Dir(protocBinPath), "..", "include"))
	if err != nil {
		return nil, err
	}
	wktFileInfo, err := os.Stat(filepath.Join(protocIncludePath, "google", "protobuf", "any.proto"))
	if err != nil {
		return nil, err
	}
	if !wktFileInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("could not find google/protobuf/any.proto in %s", protocIncludePath)
	}

	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	tempFilePath := tempFile.Name()
	defer func() {
		if err := os.Remove(tempFilePath); err != nil && retErr == nil {
			retErr = err
		}
	}()

	args := []string{"-I", protocIncludePath}
	for _, root := range roots {
		args = append(args, "-I", root)
	}
	if includeImports {
		args = append(args, "--include_imports")
	}
	if includeSourceInfo {
		args = append(args, "--include_source_info")
	}
	args = append(args, fmt.Sprintf("--descriptor_set_out=%s", tempFilePath))
	args = append(args, realFilePaths...)

	buffer := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, protocBinPath, args...)
	cmd.Env = []string{}
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	errC := make(chan error, 1)
	go func() {
		errC <- cmd.Run()
	}()
	err = nil
	select {
	case <-ctx.Done():
		_ = tempFile.Close()
		return nil, ctx.Err()
	case err = <-errC:
		if closeErr := tempFile.Close(); closeErr != nil {
			return nil, closeErr
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v returned error: %v %v", protocBinPath, args, err, buffer.String())
	}

	data, err := ioutil.ReadFile(tempFilePath)
	if err != nil {
		return nil, err
	}
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	// we do not know the FileDescriptorSet ahead of time so we cannot use it for extensions
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, fileDescriptorSet); err != nil {
		return nil, err
	}
	return fileDescriptorSet, nil
}
