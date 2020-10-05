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

package appprotogen

import (
	"context"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

// NewHandler returns a new Handler for the protogen Plugin function.
func NewHandler(f func(*protogen.Plugin) error) appproto.Handler {
	return appproto.HandlerFunc(
		func(
			ctx context.Context,
			container app.EnvStderrContainer,
			responseWriter appproto.ResponseWriter,
			request *pluginpb.CodeGeneratorRequest,
		) error {
			plugin, err := protogen.Options{}.New(request)
			if err != nil {
				return err
			}
			if err := f(plugin); err != nil {
				plugin.Error(err)
			}
			response := plugin.Response()
			for _, file := range response.File {
				if err := responseWriter.Add(file); err != nil {
					return err
				}
			}
			if errorMessage := response.GetError(); errorMessage != "" {
				if err := responseWriter.AddError(errorMessage); err != nil {
					return err
				}
			}
			responseWriter.SetFeatureProto3Optional()
			return nil
		},
	)
}
