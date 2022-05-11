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

// Code generated by protoc-gen-go-apiclientconnect. DO NOT EDIT.

package registryv1alpha1apiclientconnect

import (
	context "context"
	registryv1alpha1connect "github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	v1alpha11 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
)

type downloadServiceClient struct {
	logger          *zap.Logger
	client          registryv1alpha1connect.DownloadServiceClient
	contextModifier func(context.Context) context.Context
}

func NewDownloadServiceClient(
	httpClient connect_go.HTTPClient,
	address string,
	contextModifier func(context.Context) context.Context,
	options ...connect_go.ClientOption,
) *downloadServiceClient {
	return &downloadServiceClient{
		client: registryv1alpha1connect.NewDownloadServiceClient(
			httpClient,
			address,
			options...,
		),
		contextModifier: contextModifier,
	}
}

// Download downloads.
func (s *downloadServiceClient) Download(
	ctx context.Context,
	owner string,
	repository string,
	reference string,
) (module *v1alpha1.Module, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.Download(
		ctx,
		connect_go.NewRequest(
			&v1alpha11.DownloadRequest{
				Owner:      owner,
				Repository: repository,
				Reference:  reference,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.Module, nil
}
