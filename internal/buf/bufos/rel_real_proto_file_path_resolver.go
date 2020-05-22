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
	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
)

type relRealProtoFilePathResolver struct {
	chainedResolver   bufbuild.ProtoRealFilePathResolver
	fetchPathResolver buffetch.PathResolver
}

// newRelRealProtoFilePathResolver returns a new ProtoRealFilePathResolver that will:
//
// - Apply the chained resolver, if it is not nil.
// - Apply the path resolver.
func newRelRealProtoFilePathResolver(
	chainedResolver bufbuild.ProtoRealFilePathResolver,
	fetchPathResolver buffetch.PathResolver,
) *relRealProtoFilePathResolver {
	return &relRealProtoFilePathResolver{
		chainedResolver:   chainedResolver,
		fetchPathResolver: fetchPathResolver,
	}
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
	return p.fetchPathResolver.RelPathToExternalPath(inputFilePath)
}
