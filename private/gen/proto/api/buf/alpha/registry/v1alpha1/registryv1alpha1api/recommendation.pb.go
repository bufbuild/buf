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

// RecommendationService is the recommendation service.
type RecommendationService interface {
	// RecommendedRepositories returns a list of recommended repositories.
	RecommendedRepositories(ctx context.Context) (repositories []*v1alpha1.RecommendedRepository, err error)
	// RecommendedTemplates returns a list of recommended templates.
	RecommendedTemplates(ctx context.Context) (templates []*v1alpha1.RecommendedTemplate, err error)
	// ListRecommendedRepositories returns a list of recommended repositories that user have access to.
	ListRecommendedRepositories(ctx context.Context) (repositories []*v1alpha1.RecommendedRepository, err error)
	// ListRecommendedTemplates returns a list of recommended templates that user have access to.
	ListRecommendedTemplates(ctx context.Context) (templates []*v1alpha1.RecommendedTemplate, err error)
	// SetRecommendedRepositories set the list of repository recommendations in the server.
	SetRecommendedRepositories(ctx context.Context, repositories []*v1alpha1.RecommendingRepository) (err error)
	// SetRecommendedTemplates set the list of template recommendations in the server.
	SetRecommendedTemplates(ctx context.Context, templates []*v1alpha1.RecommendingTemplate) (err error)
}
