// Copyright 2020-2021 Buf Technologies, Inc.
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
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type userService struct {
	logger          *zap.Logger
	client          v1alpha1.UserServiceClient
	contextModifier func(context.Context) context.Context
}

// CreateUser creates a new user with the given username.
func (s *userService) CreateUser(ctx context.Context, username string) (user *v1alpha1.User, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.CreateUser(
		ctx,
		&v1alpha1.CreateUserRequest{
			Username: username,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.User, nil
}

// GetUser gets a user by ID.
func (s *userService) GetUser(ctx context.Context, id string) (user *v1alpha1.User, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetUser(
		ctx,
		&v1alpha1.GetUserRequest{
			Id: id,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.User, nil
}

// GetUserByUsername gets a user by username.
func (s *userService) GetUserByUsername(ctx context.Context, username string) (user *v1alpha1.User, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetUserByUsername(
		ctx,
		&v1alpha1.GetUserByUsernameRequest{
			Username: username,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.User, nil
}

// ListUsers lists all users.
func (s *userService) ListUsers(
	ctx context.Context,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (users []*v1alpha1.User, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListUsers(
		ctx,
		&v1alpha1.ListUsersRequest{
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Users, response.NextPageToken, nil
}

// ListOrganizationUsers lists all users for an organization.
// TODO: move this to organization service
func (s *userService) ListOrganizationUsers(
	ctx context.Context,
	organizationId string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (users []*v1alpha1.OrganizationUser, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListOrganizationUsers(
		ctx,
		&v1alpha1.ListOrganizationUsersRequest{
			OrganizationId: organizationId,
			PageSize:       pageSize,
			PageToken:      pageToken,
			Reverse:        reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Users, response.NextPageToken, nil
}

// UpdateUserUsername updates a user's username.
func (s *userService) UpdateUserUsername(ctx context.Context, newUsername string) (user *v1alpha1.User, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.UpdateUserUsername(
		ctx,
		&v1alpha1.UpdateUserUsernameRequest{
			NewUsername: newUsername,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.User, nil
}

// DeleteUser deletes a user.
func (s *userService) DeleteUser(ctx context.Context) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeleteUser(
		ctx,
		&v1alpha1.DeleteUserRequest{},
	)
	if err != nil {
		return err
	}
	return nil
}

// Deactivate user deactivates a user.
func (s *userService) DeactivateUser(ctx context.Context, id string) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeactivateUser(
		ctx,
		&v1alpha1.DeactivateUserRequest{
			Id: id,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// AddUserOrganizationScopeByName adds an organization scope for a specific organization to a user by name.
func (s *userService) AddUserOrganizationScopeByName(
	ctx context.Context,
	name string,
	organizationName string,
	organizationScope v1alpha1.OrganizationScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddUserOrganizationScopeByName(
		ctx,
		&v1alpha1.AddUserOrganizationScopeByNameRequest{
			Name:              name,
			OrganizationName:  organizationName,
			OrganizationScope: organizationScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveUserOrganizationScope removes an organization scope for a specific organization from a user by ID.
func (s *userService) RemoveUserOrganizationScope(
	ctx context.Context,
	id string,
	organizationId string,
	organizationScope v1alpha1.OrganizationScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveUserOrganizationScope(
		ctx,
		&v1alpha1.RemoveUserOrganizationScopeRequest{
			Id:                id,
			OrganizationId:    organizationId,
			OrganizationScope: organizationScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// UpdateUserServerRole update the role of an user in the server.
func (s *userService) UpdateUserServerRole(
	ctx context.Context,
	userId string,
	serverRole v1alpha1.ServerRole,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.UpdateUserServerRole(
		ctx,
		&v1alpha1.UpdateUserServerRoleRequest{
			UserId:     userId,
			ServerRole: serverRole,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
