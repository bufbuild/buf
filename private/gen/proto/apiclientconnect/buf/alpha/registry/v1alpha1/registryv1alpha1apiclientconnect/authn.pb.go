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
	registryv1alpha1api "github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha1/registryv1alpha1api"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
)

type authnService struct {
	logger          *zap.Logger
	client          registryv1alpha1api.AuthnService
	contextModifier func(context.Context) context.Context
}

// GetCurrentUser gets information associated with the current user.
//
// The user's ID is retrieved from the request's authentication header.
func (s *authnService) GetCurrentUser(ctx context.Context) (user *v1alpha1.User, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetCurrentUser(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.GetCurrentUserRequest{}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.User, nil
}

// GetCurrentUserSubject gets the currently logged in users subject.
//
// The user's ID is retrieved from the request's authentication header.
func (s *authnService) GetCurrentUserSubject(ctx context.Context) (subject string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetCurrentUserSubject(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.GetCurrentUserSubjectRequest{}),
	)
	if err != nil {
		return "", err
	}
	return response.Msg.Subject, nil
}
