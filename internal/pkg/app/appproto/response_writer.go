// Copyright 2020 Buf Technologies, Inc.
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

package appproto

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

type responseWriter struct {
	files                 []*pluginpb.CodeGeneratorResponse_File
	fileNames             map[string]struct{}
	errorMessages         []string
	featureProto3Optional bool
	lock                  sync.RWMutex
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		fileNames: make(map[string]struct{}),
	}
}

func (r *responseWriter) Add(file *pluginpb.CodeGeneratorResponse_File) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if file == nil {
		return errors.New("add CodeGeneratorResponse.File is nil")
	}
	name := file.GetName()
	if name == "" {
		return errors.New("add CodeGeneratorResponse.File.Name is empty")
	}
	// name must be relative and not contain "." or ".." per the documentation
	normalizedName, err := normalpath.NormalizeAndValidate(name)
	if err != nil {
		return fmt.Errorf("had invalid CodeGenerator.Response.File name: %v", err)
	}
	if name != normalizedName {
		return fmt.Errorf("expected CodeGeneratorResponse.File name %s to be %s", name, normalizedName)
	}
	if _, ok := r.fileNames[name]; ok {
		return fmt.Errorf("duplicate CodeGeneratorResponse.File added: %q", name)
	}
	r.fileNames[name] = struct{}{}
	r.files = append(r.files, file)
	return nil
}

func (r *responseWriter) AddError(message string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if message == "" {
		// default to an error message to make sure we pass an error
		// if this function was called
		message = "error"
	}
	r.errorMessages = append(r.errorMessages, message)
	return nil
}

func (r *responseWriter) SetFeatureProto3Optional() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.featureProto3Optional = true
}

// should only be called once
// should be private
func (r *responseWriter) toResponse(err error) *pluginpb.CodeGeneratorResponse {
	r.lock.RLock()
	defer r.lock.RUnlock()
	response := &pluginpb.CodeGeneratorResponse{
		File: r.files,
	}
	finalErrorMessages := r.errorMessages
	if err != nil {
		errorMessage := err.Error()
		if errorMessage == "" {
			// default to an error message to make sure we pass an error
			// if err != nil
			errorMessage = "error"
		}
		finalErrorMessages = append(finalErrorMessages, errorMessage)
	}
	if len(finalErrorMessages) > 0 {
		response.Error = proto.String(strings.Join(finalErrorMessages, "\n"))
	}
	if r.featureProto3Optional {
		response.SupportedFeatures = proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL))
	}
	return response
}
