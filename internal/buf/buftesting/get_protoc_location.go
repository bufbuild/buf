package buftesting

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/errs"
)

type protocLocation struct {
	BinPath     string
	IncludePath string
}

func getProtocLocation() (*protocLocation, error) {
	binPath, err := exec.LookPath("protoc")
	if err != nil {
		return nil, err
	}
	binPath, err = filepath.Abs(binPath)
	if err != nil {
		return nil, err
	}
	includePath, err := filepath.Abs(filepath.Join(filepath.Dir(binPath), "..", "include"))
	if err != nil {
		return nil, err
	}
	wktFileInfo, err := os.Stat(filepath.Join(includePath, "google", "protobuf", "any.proto"))
	if err != nil {
		return nil, err
	}
	if !wktFileInfo.Mode().IsRegular() {
		return nil, errs.NewInternalf("could not find google/protobuf/any.proto in %s", includePath)
	}
	return &protocLocation{
		BinPath:     binPath,
		IncludePath: includePath,
	}, nil
}
