// Copyright 2020-2024 Buf Technologies, Inc.
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

package buflintplugin

import (
	"context"
	"io"

	lintv1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/plugin/lint/v1beta1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Env struct {
	Stderr io.Writer
}

type Annotation struct {
	FileName    string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	ID          string
	Message     string
}

func (a *Annotation) ToProtoAnnotation() *lintv1beta1.Annotation {
	return nil
}

func NewAnnotation(protoAnnotation *lintv1beta1.Annotation) *Annotation {
	return nil
}

func NewAnnotationForDescriptor(descriptor protoreflect.Descriptor, id string, message string) *Annotation {
	return nil
}

type File interface {
	protoreflect.FileDescriptor

	isFile()
}

type ResponseWriter interface {
	AddAnnotations(...*Annotation)
	ToProtoResponse() (*lintv1beta1.Response, error)
}

func NewResponseWriter() ResponseWriter {
	return nil
}

type Request interface {
	LintFile() []File
	AllFiles() []File
	ProtoRequest() *lintv1beta1.Request
}

func NewRequest(protoRequest *lintv1beta1.Request) (Request, error) {
	return nil, nil
}

type Handler interface {
	Handle(
		context.Context,
		Env,
		ResponseWriter,
		Request,
	) error
}
