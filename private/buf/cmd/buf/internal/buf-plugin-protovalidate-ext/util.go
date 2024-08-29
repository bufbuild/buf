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

package protovalidateextplugin

import (
	"context"

	"github.com/bufbuild/bufplugin-go/check"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func breakingRuleHandlerForFile(
	f func(
		_ context.Context,
		_ check.ResponseWriter,
		_ check.Request,
		file protoreflect.FileDescriptor,
		againstFile protoreflect.FileDescriptor,
	) error,
	checkImport bool,
) check.RuleHandler {
	return check.RuleHandlerFunc(
		func(
			ctx context.Context,
			responseWriter check.ResponseWriter,
			request check.Request,
		) error {
			filePathToFile := make(map[string]check.File)
			for _, file := range request.Files() {
				filePathToFile[file.FileDescriptor().Path()] = file
			}
			for _, againstFile := range request.AgainstFiles() {
				if !checkImport && againstFile.IsImport() {
					continue
				}
				if file, ok := filePathToFile[againstFile.FileDescriptor().Path()]; ok {
					if err := f(ctx, responseWriter, request, file.FileDescriptor(), againstFile.FileDescriptor()); err != nil {
						return nil
					}
				}
			}
			return nil
		})
}

func breakingRuleHandlerForField(
	f func(
		_ context.Context,
		_ check.ResponseWriter,
		_ check.Request,
		field protoreflect.FieldDescriptor,
		againstField protoreflect.FieldDescriptor,
	) error,
	checkImport bool,
) check.RuleHandler {
	return breakingRuleHandlerForFile(
		func(
			ctx context.Context,
			responseWriter check.ResponseWriter,
			request check.Request,
			file protoreflect.FileDescriptor,
			againstFile protoreflect.FileDescriptor,
		) error {
			return forEachMessage(
				file.Messages(),
				againstFile.Messages(),
				func(message, againstMessage protoreflect.MessageDescriptor) error {
					return forEachField(
						message.Fields(),
						againstMessage.Fields(),
						func(field, againstField protoreflect.FieldDescriptor) error {
							return f(ctx, responseWriter, request, field, againstField)
						})
				},
			)
		},
		checkImport,
	)
}

func forEachMessage(
	messages protoreflect.MessageDescriptors,
	againstMessages protoreflect.MessageDescriptors,
	f func(message, againstMessage protoreflect.MessageDescriptor) error,
) error {
	for i := 0; i < againstMessages.Len(); i++ {
		againstMessage := againstMessages.Get(i)
		if message := messages.ByName(againstMessage.Name()); message != nil {
			if err := f(message, againstMessage); err != nil {
				return err
			}
			if err := forEachMessage(message.Messages(), againstMessage.Messages(), f); err != nil {
				return err
			}
		}
	}
	return nil
}

func forEachField(
	fields protoreflect.FieldDescriptors,
	againstFields protoreflect.FieldDescriptors,
	f func(field, againstField protoreflect.FieldDescriptor) error,
) error {
	for i := 0; i < againstFields.Len(); i++ {
		againstField := againstFields.Get(i)
		if field := fields.ByName(againstField.Name()); field != nil {
			if err := f(field, againstField); err != nil {
				return err
			}
		}
	}
	return nil
}
