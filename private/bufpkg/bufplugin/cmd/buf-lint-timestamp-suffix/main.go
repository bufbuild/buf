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

package main

import (
	"context"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/buflintplugin"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const id = "TIMESTAMP_SUFFIX"

func main() {
	buflintplugin.Main(buflintplugin.HandlerFunc(handle))
}

func handle(
	ctx context.Context,
	env buflintplugin.Env,
	responseWriter buflintplugin.ResponseWriter,
	request buflintplugin.Request,
) error {
	for _, file := range request.LintFiles() {
		messages := file.Messages()
		for i := 0; i < messages.Len(); i++ {
			if err := handleMessageDescriptor(responseWriter, messages.Get(i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func handleMessageDescriptor(
	responseWriter buflintplugin.ResponseWriter,
	messageDescriptor protoreflect.MessageDescriptor,
) error {
	fields := messageDescriptor.Fields()
	for i := 0; i < fields.Len(); i++ {
		if err := handleFieldDescriptor(responseWriter, fields.Get(i)); err != nil {
			return err
		}
	}

	messages := messageDescriptor.Messages()
	for i := 0; i < messages.Len(); i++ {
		if err := handleMessageDescriptor(responseWriter, messages.Get(i)); err != nil {
			return err
		}
	}
	return nil
}

func handleFieldDescriptor(
	responseWriter buflintplugin.ResponseWriter,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	messageDescriptor := fieldDescriptor.Message()
	if messageDescriptor == nil {
		return nil
	}
	if string(messageDescriptor.FullName()) != "google.protobuf.Timestamp" {
		return nil
	}
	if !strings.HasSuffix(string(fieldDescriptor.Name()), "_time") {
		responseWriter.AddAnnotations(newAnnotation(fieldDescriptor))
	}
	return nil
}

func newAnnotation(descriptor protoreflect.Descriptor) *buflintplugin.Annotation {
	return buflintplugin.NewAnnotationForDescriptor(
		descriptor,
		id,
		"Fields of type google.protobuf.Timestamp must end in _time.",
	)
}
