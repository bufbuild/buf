package internal

import (
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
)

type relProtoFilePathResolver struct {
	pwd             string
	dirPath         string
	chainedResolver bufbuild.ProtoFilePathResolver
}

func newRelProtoFilePathResolver(
	dirPath string,
	chainedResolver bufbuild.ProtoFilePathResolver,
) (*relProtoFilePathResolver, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &relProtoFilePathResolver{
		pwd:             pwd,
		dirPath:         dirPath,
		chainedResolver: chainedResolver,
	}, nil
}

func (p *relProtoFilePathResolver) GetFilePath(inputFilePath string) (string, error) {
	if inputFilePath == "" {
		return "", nil
	}

	// if there is a chained resolver, apply it first
	if p.chainedResolver != nil {
		chainedFilePath, err := p.chainedResolver.GetFilePath(inputFilePath)
		if err != nil {
			if err != bufbuild.ErrFilePathUnknown {
				return "", err
			}
		} else {
			inputFilePath = chainedFilePath
		}
	}

	// if the dirPath is ".", do nothing
	if p.dirPath == "." {
		return inputFilePath, nil
	}

	// add the prefix directory
	// Normalize and Join call filepath.Clean
	inputFilePath = storagepath.Unnormalize(storagepath.Join(storagepath.Normalize(p.dirPath), storagepath.Normalize(inputFilePath)))

	// if the directory was absolute, we can output absolute paths
	if filepath.IsAbs(p.dirPath) {
		return inputFilePath, nil
	}

	absInputFilePath, err := filepath.Abs(inputFilePath)
	if err != nil {
		return "", err
	}
	return filepath.Rel(p.pwd, absInputFilePath)
}
