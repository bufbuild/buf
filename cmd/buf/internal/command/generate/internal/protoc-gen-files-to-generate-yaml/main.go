// Copyright 2020-2026 Buf Technologies, Inc.
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

// protoc-gen-files-to-generate-yaml writes one file per CodeGeneratorRequest
// listing the files to generate in that request.
//
// All files to generate in a request must be in the same directory, and the
// output is written to that directory. This is used to test that
// strategy: directory splits requests by directory, including when
// include_imports is set.
package main

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/bufbuild/protoplugin"
	"gopkg.in/yaml.v3"
)

const fileName = "files-to-generate.yaml"

func main() {
	protoplugin.Main(protoplugin.HandlerFunc(handle))
}

func handle(
	_ context.Context,
	_ protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) error {
	filesToGenerate := request.CodeGeneratorRequest().GetFileToGenerate()
	if len(filesToGenerate) == 0 {
		return errors.New("no files to generate")
	}
	var dirs []string
	for _, fileToGenerate := range filesToGenerate {
		dir := path.Dir(fileToGenerate)
		if !slices.Contains(dirs, dir) {
			dirs = append(dirs, dir)
		}
	}
	if len(dirs) > 1 {
		return fmt.Errorf("files to generate span multiple directories: %s", strings.Join(dirs, ", "))
	}
	data, err := yaml.Marshal(&externalFile{Files: filesToGenerate})
	if err != nil {
		return err
	}
	responseWriter.AddFile(path.Join(dirs[0], fileName), string(data))
	return nil
}

type externalFile struct {
	Files []string `json:"files,omitempty" yaml:"files,omitempty"`
}
