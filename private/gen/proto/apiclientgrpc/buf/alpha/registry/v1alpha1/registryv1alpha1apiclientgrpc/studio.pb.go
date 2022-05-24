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

type studioService struct {
	logger          *zap.Logger
	client          v1alpha1.StudioServiceClient
	contextModifier func(context.Context) context.Context
}

// ListStudioAgentPresets returns a list of agent presets in the server.
func (s *studioService) ListStudioAgentPresets(ctx context.Context) (agents []*v1alpha1.StudioAgentPreset, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListStudioAgentPresets(
		ctx,
		&v1alpha1.ListStudioAgentPresetsRequest{},
	)
	if err != nil {
		return nil, err
	}
	return response.Agents, nil
}

// SetStudioAgentPresets sets the list of agent presets in the server.
func (s *studioService) SetStudioAgentPresets(ctx context.Context, agents []*v1alpha1.StudioAgentPreset) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.SetStudioAgentPresets(
		ctx,
		&v1alpha1.SetStudioAgentPresetsRequest{
			Agents: agents,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
