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

// Code generated by protoc-gen-go-api. DO NOT EDIT.

package registryv1alpha1api

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

// StudioRequestService exposes the functionality to manage favorited Requests
// from Studio.
type StudioRequestService interface {
	// CreateStudioRequest registers a favorite Studio Requests to the caller's
	// BSR profile.
	CreateStudioRequest(
		ctx context.Context,
		repositoryId string,
		name string,
		targetBaseUrl string,
		service string,
		method string,
		body string,
		headers map[string]string,
		includeCookies bool,
		protocol v1alpha1.StudioProtocol,
		agentUrl string,
	) (createdRequest *v1alpha1.StudioRequest, err error)
	// RenameStudioRequest renames an existing Studio Request.
	RenameStudioRequest(
		ctx context.Context,
		id string,
		newName string,
	) (renamedRequest *v1alpha1.StudioRequest, err error)
	// DeleteStudioRequest removes a favorite Studio Request from the caller's BSR
	// profile.
	DeleteStudioRequest(ctx context.Context, id string) (err error)
	// ListStudioRequests shows the caller's favorited Studio Requests.
	ListStudioRequests(
		ctx context.Context,
		pageToken string,
	) (requests []*v1alpha1.StudioRequest, nextPageToken string, err error)
}
