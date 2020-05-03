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

package bufos

import (
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
)

type relRealProtoFilePathResolver struct {
	pwd             string
	dirPath         string
	chainedResolver bufbuild.ProtoRealFilePathResolver
}

// newRelRealProtoFilePathResolver returns a new ProtoRealFilePathResolver that will:
//
// - Apply the chained resolver, if it is not nil.
// - Add the dirPath as a prefix.
// - Make the path relative to pwd if the path is relative, or return the path if it is absolute.
func newRelRealProtoFilePathResolver(
	dirPath string,
	chainedResolver bufbuild.ProtoRealFilePathResolver,
) (*relRealProtoFilePathResolver, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &relRealProtoFilePathResolver{
		pwd:             pwd,
		dirPath:         dirPath,
		chainedResolver: chainedResolver,
	}, nil
}

func (p *relRealProtoFilePathResolver) GetRealFilePath(inputFilePath string) (string, error) {
	if inputFilePath == "" {
		return "", nil
	}

	// if there is a chained resolver, apply it first
	if p.chainedResolver != nil {
		chainedFilePath, err := p.chainedResolver.GetRealFilePath(inputFilePath)
		if err != nil {
			return "", err
		} else if chainedFilePath != "" {
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
