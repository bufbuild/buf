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

// Code generated by protoc-gen-go-apiclientgrpc. DO NOT EDIT.

package registryv1alpha1apiclientgrpc

import (
	context "context"
	v1alpha11 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/convert/v1alpha1"
	v1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type convertService struct {
	logger          *zap.Logger
	client          v1alpha1.ConvertServiceClient
	contextModifier func(context.Context) context.Context
}

// Convert converts a serialized message according to
// the provided type name using either an image or a module_info.
func (s *convertService) Convert(
	ctx context.Context,
	typeName string,
	image *v1.Image,
	moduleInfo *v1alpha11.ModuleInfo,
	messageBytes []byte,
	inputFormat v1alpha1.ConvertFormat,
	outputFormat v1alpha1.ConvertFormat,
) (result []byte, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.Convert(
		ctx,
		&v1alpha1.ConvertRequest{
			TypeName:     typeName,
			Image:        image,
			ModuleInfo:   moduleInfo,
			MessageBytes: messageBytes,
			InputFormat:  inputFormat,
			OutputFormat: outputFormat,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}
