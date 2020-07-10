// Copyright 2020 Buf Technologies, Inc.
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

package prototesting

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/diff"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

// GetProtocFileDescriptorSet gets the validated FileDescriptorSet using
// protoc and the Well-Known Types on the current PATH.
//
// Only use for testing.
func GetProtocFileDescriptorSet(
	ctx context.Context,
	roots []string,
	realFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
	experimentalAllowProto3Optional bool,
) (_ *descriptorpb.FileDescriptorSet, retErr error) {
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

	if err := RunProtoc(
		ctx,
		roots,
		realFilePaths,
		includeImports,
		includeSourceInfo,
		experimentalAllowProto3Optional,
		nil,
		nil,
		fmt.Sprintf("--descriptor_set_out=%s", tempFilePath),
	); err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(tempFilePath)
	if err != nil {
		return nil, err
	}
	firstFileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, firstFileDescriptorSet); err != nil {
		return nil, err
	}
	resolver, err := protoencoding.NewResolver(
		firstFileDescriptorSet.File...,
	)
	if err != nil {
		return nil, err
	}
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := protoencoding.NewWireUnmarshaler(resolver).Unmarshal(data, fileDescriptorSet); err != nil {
		return nil, err
	}
	return fileDescriptorSet, nil
}

// RunProtoc runs protoc.
func RunProtoc(
	ctx context.Context,
	roots []string,
	realFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
	experimentalAllowProto3Optional bool,
	env map[string]string,
	stdout io.Writer,
	extraFlags ...string,
) error {
	protocBinPath, err := exec.LookPath("protoc")
	if err != nil {
		return err
	}
	protocBinPath, err = filepath.Abs(protocBinPath)
	if err != nil {
		return err
	}
	protocIncludePath, err := filepath.Abs(filepath.Join(filepath.Dir(protocBinPath), "..", "include"))
	if err != nil {
		return err
	}
	wktFileInfo, err := os.Stat(filepath.Join(protocIncludePath, "google", "protobuf", "any.proto"))
	if err != nil {
		return err
	}
	if !wktFileInfo.Mode().IsRegular() {
		return fmt.Errorf("could not find google/protobuf/any.proto in %s", protocIncludePath)
	}

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
	if experimentalAllowProto3Optional {
		args = append(args, "--experimental_allow_proto3_optional")
	}
	args = append(args, extraFlags...)
	args = append(args, realFilePaths...)

	stderr := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, protocBinPath, args...)
	var environ []string
	for key, value := range env {
		environ = append(environ, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = environ
	if stdout == nil {
		cmd.Stdout = stderr
	} else {
		cmd.Stdout = stdout
	}
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s returned error: %v %v", protocBinPath, err, stderr.String())
	}
	return nil
}

// DiffFileDescriptorSetsWire diffs the two FileDescriptorSets using proto.MarshalWire.
func DiffFileDescriptorSetsWire(
	one *descriptorpb.FileDescriptorSet,
	two *descriptorpb.FileDescriptorSet,
	oneName string,
	twoName string,
) (string, error) {
	oneData, err := protoencoding.NewWireMarshaler().Marshal(one)
	if err != nil {
		return "", err
	}
	twoData, err := protoencoding.NewWireMarshaler().Marshal(two)
	if err != nil {
		return "", err
	}
	output, err := diff.Diff(oneData, twoData, oneName, twoName, false)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// DiffFileDescriptorSetsJSON diffs the two FileDescriptorSets using JSON.
func DiffFileDescriptorSetsJSON(
	one *descriptorpb.FileDescriptorSet,
	two *descriptorpb.FileDescriptorSet,
	oneName string,
	twoName string,
) (string, error) {
	oneResolver, err := protoencoding.NewResolver(one.File...)
	if err != nil {
		return "", err
	}
	oneData, err := protoencoding.NewJSONMarshalerIndent(oneResolver).Marshal(one)
	if err != nil {
		return "", err
	}
	twoResolver, err := protoencoding.NewResolver(two.File...)
	if err != nil {
		return "", err
	}
	twoData, err := protoencoding.NewJSONMarshalerIndent(twoResolver).Marshal(two)
	if err != nil {
		return "", err
	}
	output, err := diff.Diff(oneData, twoData, oneName, twoName, false)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// DiffFileDescriptorSetsCompare diffs the two FileDescriptorSets using the cmp package.
func DiffFileDescriptorSetsCompare(
	one *descriptorpb.FileDescriptorSet,
	two *descriptorpb.FileDescriptorSet,
) string {
	return cmp.Diff(one, two, protocmp.Transform())
}

// AssertFileDescriptorSetsEqual asserts that the FileDescriptorSet are equal for
// JSON and compare.
func AssertFileDescriptorSetsEqual(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	diff, err := DiffFileDescriptorSetsJSON(one, two, "buf", "protoc")
	assert.NoError(t, err)
	assert.Empty(t, diff)
	diff = DiffFileDescriptorSetsCompare(one, two)
	assert.Empty(t, diff)
}
