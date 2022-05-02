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

type searchServiceClient struct {
	client registryv1alpha1connect.SearchServiceClient
}

func newSearchServiceClient(
	httpClient connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) *searchServiceClient {
	return &searchServiceClient{
		client: registryv1alpha1connect.NewSearchServiceClient(
			httpClient,
			address,
			options...,
		),
	}
}

// Search searches the BSR.
func (s *searchServiceClient) Search(
	ctx context.Context,
	query string,
	pageSize uint32,
	pageToken uint32,
	filters []v1alpha1.SearchFilter,
) (searchResults []*v1alpha1.SearchResult, nextPageToken uint32, _ error) {
	response, err := s.client.Search(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.SearchRequest{
				Query:     query,
				PageSize:  pageSize,
				PageToken: pageToken,
				Filters:   filters,
			}),
	)
	if err != nil {
		return nil, 0, err
	}
	return response.Msg.SearchResults, response.Msg.NextPageToken, nil
}
