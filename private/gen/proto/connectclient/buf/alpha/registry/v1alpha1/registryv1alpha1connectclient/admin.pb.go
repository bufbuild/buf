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

// Code generated by protoc-gen-go-connectclient. DO NOT EDIT.

package registryv1alpha1connectclient

import (
	context "context"
	registryv1alpha1connect "github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
)

type adminServiceClient struct {
	client registryv1alpha1connect.AdminServiceClient
}

func newAdminServiceClient(
	httpClient connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) *adminServiceClient {
	return &adminServiceClient{
		client: registryv1alpha1connect.NewAdminServiceClient(
			httpClient,
			address,
			options...,
		),
	}
}

// ForceDeleteUser forces to delete a user. Resources and organizations that are
// solely owned by the user will also be deleted.
func (s *adminServiceClient) ForceDeleteUser(
	ctx context.Context,
	userId string,
) (user *v1alpha1.User, organizations []*v1alpha1.Organization, repositories []*v1alpha1.Repository, plugins []*v1alpha1.Plugin, templates []*v1alpha1.Template, _ error) {
	response, err := s.client.ForceDeleteUser(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.ForceDeleteUserRequest{
				UserId: userId,
			}),
	)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return response.Msg.User, response.Msg.Organizations, response.Msg.Repositories, response.Msg.Plugins, response.Msg.Templates, nil
}
