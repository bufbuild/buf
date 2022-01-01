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
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type recommendationService struct {
	logger          *zap.Logger
	client          v1alpha1.RecommendationServiceClient
	contextModifier func(context.Context) context.Context
}

// RecommendedRepositories returns a list of recommended repositories.
func (s *recommendationService) RecommendedRepositories(ctx context.Context) (repositories []*v1alpha1.RecommendedRepository, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.RecommendedRepositories(
		ctx,
		&v1alpha1.RecommendedRepositoriesRequest{},
	)
	if err != nil {
		return nil, err
	}
	return response.Repositories, nil
}

// RecommendedTemplates returns a list of recommended templates.
func (s *recommendationService) RecommendedTemplates(ctx context.Context) (templates []*v1alpha1.RecommendedTemplate, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.RecommendedTemplates(
		ctx,
		&v1alpha1.RecommendedTemplatesRequest{},
	)
	if err != nil {
		return nil, err
	}
	return response.Templates, nil
}

// ListRecommendedRepositories returns a list of recommended repositories that user have access to.
func (s *recommendationService) ListRecommendedRepositories(ctx context.Context) (repositories []*v1alpha1.RecommendedRepository, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListRecommendedRepositories(
		ctx,
		&v1alpha1.ListRecommendedRepositoriesRequest{},
	)
	if err != nil {
		return nil, err
	}
	return response.Repositories, nil
}

// ListRecommendedTemplates returns a list of recommended templates that user have access to.
func (s *recommendationService) ListRecommendedTemplates(ctx context.Context) (templates []*v1alpha1.RecommendedTemplate, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListRecommendedTemplates(
		ctx,
		&v1alpha1.ListRecommendedTemplatesRequest{},
	)
	if err != nil {
		return nil, err
	}
	return response.Templates, nil
}

// SetRecommendedRepositories set the list of repository recommendations in the server.
func (s *recommendationService) SetRecommendedRepositories(ctx context.Context, repositories []*v1alpha1.SetRecommendedRepository) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.SetRecommendedRepositories(
		ctx,
		&v1alpha1.SetRecommendedRepositoriesRequest{
			Repositories: repositories,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// SetRecommendedTemplates set the list of template recommendations in the server.
func (s *recommendationService) SetRecommendedTemplates(ctx context.Context, templates []*v1alpha1.SetRecommendedTemplate) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.SetRecommendedTemplates(
		ctx,
		&v1alpha1.SetRecommendedTemplatesRequest{
			Templates: templates,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
