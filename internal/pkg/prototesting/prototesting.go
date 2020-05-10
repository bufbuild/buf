package prototesting

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"github.com/bufbuild/buf/internal/pkg/util/utildiff"
	"google.golang.org/protobuf/types/descriptorpb"
)

// DiffFileDescriptorSetsWire diffs the two FileDescriptorSets using proto.MarshalWire.
func DiffFileDescriptorSetsWire(one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet, name string) (string, error) {
	oneData, err := protoencoding.NewWireMarshaler().Marshal(one)
	if err != nil {
		return "", err
	}
	twoData, err := protoencoding.NewWireMarshaler().Marshal(two)
	if err != nil {
		return "", err
	}
	output, err := utildiff.Diff(oneData, twoData, name)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// DiffFileDescriptorSetsJSON diffs the two FileDescriptorSets using JSON.
//
// TODO: this does NOT use any resolver, so extensions will be dropped. This needs to be updated.
func DiffFileDescriptorSetsJSON(one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet, name string) (string, error) {
	oneData, err := protoencoding.NewJSONMarshalerIndent(nil).Marshal(one)
	if err != nil {
		return "", err
	}
	twoData, err := protoencoding.NewJSONMarshalerIndent(nil).Marshal(two)
	if err != nil {
		return "", err
	}
	output, err := utildiff.Diff(oneData, twoData, name)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

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
