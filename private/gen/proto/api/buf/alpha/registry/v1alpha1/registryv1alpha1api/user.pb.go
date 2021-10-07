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

// Code generated by protoc-gen-go-api. DO NOT EDIT.

package registryv1alpha1api

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

// UserService is the User service.
type UserService interface {
	// CreateUser creates a new user with the given username.
	CreateUser(ctx context.Context, username string) (user *v1alpha1.User, err error)
	// GetUser gets a user by ID.
	GetUser(ctx context.Context, id string) (user *v1alpha1.User, err error)
	// GetUserByUsername gets a user by username.
	GetUserByUsername(ctx context.Context, username string) (user *v1alpha1.User, err error)
	// ListUsers lists all users.
	ListUsers(
		ctx context.Context,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (users []*v1alpha1.User, nextPageToken string, err error)
	// ListOrganizationUsers lists all users for an organization.
	ListOrganizationUsers(
		ctx context.Context,
		organizationId string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (users []*v1alpha1.User, nextPageToken string, err error)
	// UpdateUserUsername updates a user's username.
	UpdateUserUsername(ctx context.Context, newUsername string) (user *v1alpha1.User, err error)
	// DeleteUser deletes a user.
	DeleteUser(ctx context.Context) (err error)
	// Deactivate user deactivates a user.
	DeactivateUser(ctx context.Context, id string) (err error)
	// AddUserOrganizationScopeByName adds an organization scope for a specific organization to a user by name.
	AddUserOrganizationScopeByName(
		ctx context.Context,
		name string,
		organizationName string,
		organizationScope v1alpha1.OrganizationScope,
	) (err error)
	// RemoveUserOrganizationScope removes an organization scope for a specific organization from a user by ID.
	RemoveUserOrganizationScope(
		ctx context.Context,
		id string,
		organizationId string,
		organizationScope v1alpha1.OrganizationScope,
	) (err error)
	// RemoveUserOrganizationScopeByName removes an organization scope for a specific organization from a user by name.
	RemoveUserOrganizationScopeByName(
		ctx context.Context,
		name string,
		organizationName string,
		organizationScope v1alpha1.OrganizationScope,
	) (err error)
	// AddUserServerScope adds a server scope for a user by ID.
	AddUserServerScope(
		ctx context.Context,
		id string,
		serverScope v1alpha1.ServerScope,
	) (err error)
	// AddUserServerScopeByName adds a server scope for a user by name.
	AddUserServerScopeByName(
		ctx context.Context,
		name string,
		serverScope v1alpha1.ServerScope,
	) (err error)
	// RemoveUserServerScope removes a server scope for a user by ID.
	RemoveUserServerScope(
		ctx context.Context,
		id string,
		serverScope v1alpha1.ServerScope,
	) (err error)
	// RemoveUserServerScopeByName removes a server scope for a user by name.
	RemoveUserServerScopeByName(
		ctx context.Context,
		name string,
		serverScope v1alpha1.ServerScope,
	) (err error)
}
