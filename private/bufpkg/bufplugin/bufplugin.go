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

package bufplugin

import (
	"context"
	"errors"
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
	panic("TODO")
	return nil
}

func NewAnnotation(protoAnnotation *lintv1beta1.Annotation) *Annotation {
	panic("TODO")
	return nil
}

func NewAnnotationForDescriptor(descriptor protoreflect.Descriptor, id string, message string) *Annotation {
	panic("TODO")
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
	panic("TODO")
	return nil
}

type Request interface {
	LintFiles() []File
	AllFiles() []File
	ProtoRequest() *lintv1beta1.Request
}

func NewRequest(protoRequest *lintv1beta1.Request) (Request, error) {
	return nil, errors.New("TODO")
}

type Handler interface {
	Handle(
		context.Context,
		Env,
		ResponseWriter,
		Request,
	) error
}

type HandlerFunc func(
	context.Context,
	Env,
	ResponseWriter,
	Request,
) error

func (h HandlerFunc) Handle(
	ctx context.Context,
	env Env,
	responseWriter ResponseWriter,
	request Request,
) error {
	return h(ctx, env, responseWriter, request)
}

func Main(handler Handler) {
	panic("TODO")
}

func Run(ctx context.Context, env Env, handler Handler) error {
	return errors.New("TODO")
}
