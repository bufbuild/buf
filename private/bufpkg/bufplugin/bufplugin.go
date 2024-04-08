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

	pluginv1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/plugin/v1beta1"
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

func (a *Annotation) ToProtoAnnotation() *pluginv1beta1.Annotation {
	return &pluginv1beta1.Annotation{
		FileName:    a.FileName,
		StartLine:   uint32(a.StartLine),
		StartColumn: uint32(a.StartColumn),
		EndLine:     uint32(a.EndLine),
		EndColumn:   uint32(a.EndColumn),
		Id:          a.ID,
		Message:     a.Message,
	}
}

func NewAnnotation(protoAnnotation *pluginv1beta1.Annotation) *Annotation {
	return &Annotation{
		FileName:    protoAnnotation.FileName,
		StartLine:   int(protoAnnotation.StartLine),
		StartColumn: int(protoAnnotation.StartColumn),
		EndLine:     int(protoAnnotation.EndLine),
		EndColumn:   int(protoAnnotation.EndColumn),
		ID:          protoAnnotation.Id,
		Message:     protoAnnotation.Message,
	}
}

func NewAnnotationForDescriptor(descriptor protoreflect.Descriptor, id string, message string) *Annotation {
	annotation := &Annotation{
		ID:      id,
		Message: message,
	}
	fileDescriptor := descriptor.ParentFile()
	if fileDescriptor == nil {
		// ParentFile is documented to maybe be nil for some reason.
		return annotation
	}
	annotation.FileName = fileDescriptor.Path()
	sourceLocation := fileDescriptor.SourceLocations().ByDescriptor(descriptor)
	// TODO: The protoreflect API is a disaster. It says that "If there is no SourceLocation,
	// the zero value is returned", but equality is not easy because SourceLocation contains
	// a slice. This is just a mess. Also need to reconcile the zero-indexing.
	_ = sourceLocation
	panic("TODO")
	return annotation
}

type File interface {
	protoreflect.FileDescriptor

	isFile()
}

type LintResponseWriter interface {
	AddAnnotations(...*Annotation)
	ToProtoLintResponse() (*pluginv1beta1.LintResponse, error)

	isLintResponseWriter()
}

func NewLintResponseWriter() LintResponseWriter {
	panic("TODO")
	return nil
}

type LintRequest interface {
	LintFiles() []File
	AllFiles() []File
	ProtoLintRequest() *pluginv1beta1.LintRequest

	isLintRequest()
}

func NewLintRequest(protoLintRequest *pluginv1beta1.LintRequest) (LintRequest, error) {
	return nil, errors.New("TODO")
}

type LintHandler interface {
	Handle(
		context.Context,
		Env,
		LintResponseWriter,
		LintRequest,
	) error
}

type LintHandlerFunc func(
	context.Context,
	Env,
	LintResponseWriter,
	LintRequest,
) error

func (l LintHandlerFunc) Handle(
	ctx context.Context,
	env Env,
	responseWriter LintResponseWriter,
	request LintRequest,
) error {
	return l(ctx, env, responseWriter, request)
}

func LintMain(handler LintHandler) {
	panic("TODO")
}

func LintRun(ctx context.Context, env Env, handler LintHandler) error {
	return errors.New("TODO")
}
