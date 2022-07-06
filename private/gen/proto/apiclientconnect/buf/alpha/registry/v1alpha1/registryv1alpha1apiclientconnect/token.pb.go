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
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type tokenServiceClient struct {
	logger *zap.Logger
	client registryv1alpha1connect.TokenServiceClient
}

// CreateToken creates a new token suitable for machine-to-machine authentication.
func (s *tokenServiceClient) CreateToken(
	ctx context.Context,
	note string,
	expireTime *timestamppb.Timestamp,
) (token string, _ error) {
	response, err := s.client.CreateToken(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.CreateTokenRequest{
				Note:       note,
				ExpireTime: expireTime,
			}),
	)
	if err != nil {
		return "", err
	}
	return response.Msg.Token, nil
}

// GetToken gets the specific token for the user
//
// This method requires authentication.
func (s *tokenServiceClient) GetToken(ctx context.Context, tokenId string) (token *v1alpha1.Token, _ error) {
	response, err := s.client.GetToken(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.GetTokenRequest{
				TokenId: tokenId,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.Token, nil
}

// ListTokens lists the users active tokens
//
// This method requires authentication.
func (s *tokenServiceClient) ListTokens(
	ctx context.Context,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (tokens []*v1alpha1.Token, nextPageToken string, _ error) {
	response, err := s.client.ListTokens(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.ListTokensRequest{
				PageSize:  pageSize,
				PageToken: pageToken,
				Reverse:   reverse,
			}),
	)
	if err != nil {
		return nil, "", err
	}
	return response.Msg.Tokens, response.Msg.NextPageToken, nil
}

// DeleteToken deletes an existing token.
//
// This method requires authentication.
func (s *tokenServiceClient) DeleteToken(ctx context.Context, tokenId string) (_ error) {
	_, err := s.client.DeleteToken(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.DeleteTokenRequest{
				TokenId: tokenId,
			}),
	)
	if err != nil {
		return err
	}
	return nil
}
