// Copyright 2020-2022 Buf Technologies, Inc.
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

package appprotoexec

import (
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
)

type templateData struct {
	// Filename is the recommended values for output files. It is the file's
	// name without the .proto extension.
	Filename string
	// GoPackageName is the file's package represented as a Go package identifier.
	GoPackageName string
	// File is the primary source of input used in the template.
	File *descriptorpb.FileDescriptorProto
}

func newTemplateData(file *descriptorpb.FileDescriptorProto) *templateData {
	return &templateData{
		Filename: strings.TrimSuffix(file.GetName(), ".proto"),
		// TODO: The goPackage function should actually consider the entire go_package path.
		// If the go_package contains a ';' component, it should be used.
		GoPackageName: strings.Replace(file.GetPackage(), ".", "", -1),
		File:          file,
	}
}
