// Copyright 2020-2023 Buf Technologies, Inc.
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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/diff"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

// GetProtocFileDescriptorSet gets the validated FileDescriptorSet using
// protoc and the Well-Known Types on the current PATH.
//
// Only use for testing.
func GetProtocFileDescriptorSet(
	ctx context.Context,
	runner command.Runner,
	roots []string,
	realFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) (_ *descriptorpb.FileDescriptorSet, retErr error) {
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}
	tempFilePath := tempFile.Name()
	defer func() {
		if err := tempFile.Close(); err != nil {
			retErr = multierr.Append(retErr, err)
		}
		if err := os.Remove(tempFilePath); err != nil {
			retErr = multierr.Append(retErr, err)
		}
	}()

	if err := RunProtoc(
		ctx,
		runner,
		roots,
		realFilePaths,
		includeImports,
		includeSourceInfo,
		nil,
		nil,
		fmt.Sprintf("--descriptor_set_out=%s", tempFilePath),
	); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tempFilePath)
	if err != nil {
		return nil, err
	}
	firstFileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, firstFileDescriptorSet); err != nil {
		return nil, err
	}
	resolver, err := protoencoding.NewResolver(protodescriptor.FileDescriptorsForFileDescriptorSet(firstFileDescriptorSet)...)
	if err != nil {
		return nil, err
	}
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := protoencoding.NewWireUnmarshaler(resolver).Unmarshal(data, fileDescriptorSet); err != nil {
		return nil, err
	}
	return fileDescriptorSet, nil
}

// GetProtocVersion runs protoc --version with RunProtoc and returns the output sans the "libprotoc" prefix
func GetProtocVersion(ctx context.Context) (string, error) {
	var stdout bytes.Buffer
	runner := command.NewRunner()
	if err := RunProtoc(ctx, runner, nil, nil, false, false, nil, &stdout, "--version"); err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.TrimPrefix(stdout.String(), "libprotoc")), nil
}

// RunProtoc runs protoc.
func RunProtoc(
	ctx context.Context,
	runner command.Runner,
	roots []string,
	realFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
	env map[string]string,
	stdout io.Writer,
	extraFlags ...string,
) error {
	protocBinPath, err := getProtocBinPath()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(protocBinPath); err == nil {
		protocBinPath = resolved
	}
	protocIncludePath, err := getProtocIncludePath(protocBinPath)
	if err != nil {
		return err
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
	args = append(args, extraFlags...)
	args = append(args, realFilePaths...)

	stderr := bytes.NewBuffer(nil)
	if stdout == nil {
		stdout = stderr
	}
	if err := runner.Run(
		ctx,
		protocBinPath,
		command.RunWithArgs(args...),
		command.RunWithEnv(env),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
	); err != nil {
		return fmt.Errorf("%s returned error: %v %v", protocBinPath, err, stderr.String())
	}
	return nil
}

// DiffFileDescriptorSetsWire diffs the two FileDescriptorSets using proto.MarshalWire.
func DiffFileDescriptorSetsWire(
	ctx context.Context,
	runner command.Runner,
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
	output, err := diff.Diff(ctx, runner, oneData, twoData, oneName, twoName)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// DiffFileDescriptorSetsJSON diffs the two FileDescriptorSets using JSON.
func DiffFileDescriptorSetsJSON(
	ctx context.Context,
	runner command.Runner,
	one *descriptorpb.FileDescriptorSet,
	two *descriptorpb.FileDescriptorSet,
	oneName string,
	twoName string,
) (string, error) {
	oneResolver, err := protoencoding.NewResolver(protodescriptor.FileDescriptorsForFileDescriptorSet(one)...)
	if err != nil {
		return "", err
	}
	oneData, err := protoencoding.NewJSONMarshaler(oneResolver, protoencoding.JSONMarshalerWithIndent()).Marshal(one)
	if err != nil {
		return "", err
	}
	twoResolver, err := protoencoding.NewResolver(protodescriptor.FileDescriptorsForFileDescriptorSet(two)...)
	if err != nil {
		return "", err
	}
	twoData, err := protoencoding.NewJSONMarshaler(twoResolver, protoencoding.JSONMarshalerWithIndent()).Marshal(two)
	if err != nil {
		return "", err
	}
	output, err := diff.Diff(ctx, runner, oneData, twoData, oneName, twoName)
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
func AssertFileDescriptorSetsEqual(
	t *testing.T,
	runner command.Runner,
	one *descriptorpb.FileDescriptorSet,
	two *descriptorpb.FileDescriptorSet,
) {
	diff, err := DiffFileDescriptorSetsJSON(context.Background(), runner, one, two, "buf", "protoc")
	assert.NoError(t, err)
	assert.Empty(t, diff)
	diff = DiffFileDescriptorSetsCompare(one, two)
	assert.Empty(t, diff)
}

// getProtocBinPath gets the os-specific path for the protoc binary on disk.
func getProtocBinPath() (string, error) {
	protocBinPath, err := exec.LookPath("protoc")
	if err != nil {
		return "", err
	}
	return filepath.Abs(protocBinPath)
}

// checkWKT checks that the well-known types are in included in the protoc libraries and returns an error if
// they are not present.
func checkWKT(protocIncludePath string) error {
	// OK to use os.Stat here
	wktFileInfo, err := os.Stat(filepath.Join(protocIncludePath, "google", "protobuf", "any.proto"))
	if err != nil {
		return err
	}
	if !wktFileInfo.Mode().IsRegular() {
		return fmt.Errorf("could not find google/protobuf/any.proto in %s", protocIncludePath)
	}
	return nil
}
