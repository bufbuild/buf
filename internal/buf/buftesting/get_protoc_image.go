package buftesting

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/errs"
)

func getProtocImage(
	ctx context.Context,
	protocLocation *protocLocation,
	roots []string,
	realFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) (_ bufpb.Image, retErr error) {
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

	args := []string{
		"-I",
		protocLocation.IncludePath,
	}
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
	cmd := exec.CommandContext(ctx, protocLocation.BinPath, args...)
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
		return nil, errs.NewInternalf("%s %v returned error: %v %v", protocLocation.BinPath, args, err, buffer.String())
	}

	data, err := ioutil.ReadFile(tempFilePath)
	if err != nil {
		return nil, err
	}
	return bufpb.UnmarshalWireDataImage(data)
}
