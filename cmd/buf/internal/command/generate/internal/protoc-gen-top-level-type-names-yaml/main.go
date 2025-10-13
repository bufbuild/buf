// Copyright 2020-2025 Buf Technologies, Inc.
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

package main

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/protoplugin"
	"gopkg.in/yaml.v3"
)

const fileExt = ".top-level-type-names.yaml"

func main() {
	protoplugin.Main(protoplugin.HandlerFunc(handle))
}

func handle(
	_ context.Context,
	_ protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) error {
	fileDescriptors, err := request.FileDescriptorsToGenerate()
	if err != nil {
		return err
	}
	for _, fileDescriptor := range fileDescriptors {
		externalFile := &externalFile{}
		enumDescriptors := fileDescriptor.Enums()
		for i := range enumDescriptors.Len() {
			externalFile.Enums = append(externalFile.Enums, string(enumDescriptors.Get(i).FullName()))
		}
		messageDescriptors := fileDescriptor.Messages()
		for i := range messageDescriptors.Len() {
			externalFile.Messages = append(externalFile.Messages, string(messageDescriptors.Get(i).FullName()))
		}
		serviceDescriptors := fileDescriptor.Services()
		for i := range serviceDescriptors.Len() {
			externalFile.Services = append(externalFile.Services, string(serviceDescriptors.Get(i).FullName()))
		}
		sort.Strings(externalFile.Enums)
		sort.Strings(externalFile.Messages)
		sort.Strings(externalFile.Services)
		data, err := yaml.Marshal(externalFile)
		if err != nil {
			return err
		}
		responseWriter.AddFile(
			strings.TrimSuffix(fileDescriptor.Path(), filepath.Ext(filepath.FromSlash(fileDescriptor.Path())))+fileExt,
			string(data),
		)
	}
	return nil
}

type externalFile struct {
	Enums    []string `json:"enums,omitempty" yaml:"enums,omitempty"`
	Messages []string `json:"messages,omitempty" yaml:"messages,omitempty"`
	Services []string `json:"services,omitempty" yaml:"services,omitempty"`
}
