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
	v1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type teamService struct {
	logger          *zap.Logger
	client          v1alpha1.TeamServiceClient
	contextModifier func(context.Context) context.Context
}

// GetTeam gets a team by ID.
func (s *teamService) GetTeam(ctx context.Context, id string) (team *v1alpha1.Team, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetTeam(
		ctx,
		&v1alpha1.GetTeamRequest{
			Id: id,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Team, nil
}

// GetTeamByName gets a team by the combination of its name and organization.
func (s *teamService) GetTeamByName(
	ctx context.Context,
	name string,
	organizationName string,
) (team *v1alpha1.Team, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetTeamByName(
		ctx,
		&v1alpha1.GetTeamByNameRequest{
			Name:             name,
			OrganizationName: organizationName,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Team, nil
}

// ListOrganizationTeams lists all teams belonging to an organization.
func (s *teamService) ListOrganizationTeams(
	ctx context.Context,
	organizationId string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (teams []*v1alpha1.Team, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListOrganizationTeams(
		ctx,
		&v1alpha1.ListOrganizationTeamsRequest{
			OrganizationId: organizationId,
			PageSize:       pageSize,
			PageToken:      pageToken,
			Reverse:        reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Teams, response.NextPageToken, nil
}

// CreateTeam creates a new team within an organization.
func (s *teamService) CreateTeam(
	ctx context.Context,
	name string,
	organizationId string,
) (team *v1alpha1.Team, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.CreateTeam(
		ctx,
		&v1alpha1.CreateTeamRequest{
			Name:           name,
			OrganizationId: organizationId,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Team, nil
}

// CreateTeamByName creates a new team within an organization, looking up the organization by name.
func (s *teamService) CreateTeamByName(
	ctx context.Context,
	name string,
	organizationName string,
) (team *v1alpha1.Team, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.CreateTeamByName(
		ctx,
		&v1alpha1.CreateTeamByNameRequest{
			Name:             name,
			OrganizationName: organizationName,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Team, nil
}

// UpdateTeamName updates a team's name.
func (s *teamService) UpdateTeamName(
	ctx context.Context,
	id string,
	newName string,
) (team *v1alpha1.Team, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.UpdateTeamName(
		ctx,
		&v1alpha1.UpdateTeamNameRequest{
			Id:      id,
			NewName: newName,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Team, nil
}

// AddUserToTeam adds a user to a team by their respective IDs.
func (s *teamService) AddUserToTeam(
	ctx context.Context,
	id string,
	userId string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddUserToTeam(
		ctx,
		&v1alpha1.AddUserToTeamRequest{
			Id:     id,
			UserId: userId,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// AddUserToTeamByName adds a user to a team, looking up the entities by user, team, and organization names.
func (s *teamService) AddUserToTeamByName(
	ctx context.Context,
	name string,
	userName string,
	organizationName string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddUserToTeamByName(
		ctx,
		&v1alpha1.AddUserToTeamByNameRequest{
			Name:             name,
			UserName:         userName,
			OrganizationName: organizationName,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveUserFromTeam removes a user from a team by their respective IDs.
func (s *teamService) RemoveUserFromTeam(
	ctx context.Context,
	id string,
	userId string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveUserFromTeam(
		ctx,
		&v1alpha1.RemoveUserFromTeamRequest{
			Id:     id,
			UserId: userId,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveUserFromTeamByName removes a user from a team, looking up the entities by user, team, and organization names.
func (s *teamService) RemoveUserFromTeamByName(
	ctx context.Context,
	name string,
	userName string,
	organizationName string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveUserFromTeamByName(
		ctx,
		&v1alpha1.RemoveUserFromTeamByNameRequest{
			Name:             name,
			UserName:         userName,
			OrganizationName: organizationName,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// DeleteTeam deletes a team by ID.
func (s *teamService) DeleteTeam(ctx context.Context, id string) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeleteTeam(
		ctx,
		&v1alpha1.DeleteTeamRequest{
			Id: id,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// DeleteTeamByName deletes a team by the combination of its name and organization.
func (s *teamService) DeleteTeamByName(
	ctx context.Context,
	name string,
	organizationName string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeleteTeamByName(
		ctx,
		&v1alpha1.DeleteTeamByNameRequest{
			Name:             name,
			OrganizationName: organizationName,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// AddTeamOrganizationScope adds an organization scope to a team by ID.
func (s *teamService) AddTeamOrganizationScope(
	ctx context.Context,
	id string,
	organizationScope v1alpha1.OrganizationScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddTeamOrganizationScope(
		ctx,
		&v1alpha1.AddTeamOrganizationScopeRequest{
			Id:                id,
			OrganizationScope: organizationScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// AddTeamOrganizationScopeByName adds an organization scope to a team by name.
func (s *teamService) AddTeamOrganizationScopeByName(
	ctx context.Context,
	name string,
	organizationName string,
	organizationScope v1alpha1.OrganizationScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddTeamOrganizationScopeByName(
		ctx,
		&v1alpha1.AddTeamOrganizationScopeByNameRequest{
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

// RemoveTeamOrganizationScope removes an organization scope from a team by ID.
func (s *teamService) RemoveTeamOrganizationScope(
	ctx context.Context,
	id string,
	organizationScope v1alpha1.OrganizationScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveTeamOrganizationScope(
		ctx,
		&v1alpha1.RemoveTeamOrganizationScopeRequest{
			Id:                id,
			OrganizationScope: organizationScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveTeamOrganizationScopeByName removes an organization scope from a team by name.
func (s *teamService) RemoveTeamOrganizationScopeByName(
	ctx context.Context,
	name string,
	organizationName string,
	organizationScope v1alpha1.OrganizationScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveTeamOrganizationScopeByName(
		ctx,
		&v1alpha1.RemoveTeamOrganizationScopeByNameRequest{
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

// AddTeamBaseRepositoryScope adds a base repository scope to a team by ID.
func (s *teamService) AddTeamBaseRepositoryScope(
	ctx context.Context,
	id string,
	repositoryScope v1alpha1.RepositoryScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddTeamBaseRepositoryScope(
		ctx,
		&v1alpha1.AddTeamBaseRepositoryScopeRequest{
			Id:              id,
			RepositoryScope: repositoryScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// AddTeamBaseRepositoryScopeByName adds a base repository scope to a team by name.
func (s *teamService) AddTeamBaseRepositoryScopeByName(
	ctx context.Context,
	name string,
	organizationName string,
	repositoryScope v1alpha1.RepositoryScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddTeamBaseRepositoryScopeByName(
		ctx,
		&v1alpha1.AddTeamBaseRepositoryScopeByNameRequest{
			Name:             name,
			OrganizationName: organizationName,
			RepositoryScope:  repositoryScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveTeamBaseRepositoryScope removes a base repository scope from a team by ID.
func (s *teamService) RemoveTeamBaseRepositoryScope(
	ctx context.Context,
	id string,
	repositoryScope v1alpha1.RepositoryScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveTeamBaseRepositoryScope(
		ctx,
		&v1alpha1.RemoveTeamBaseRepositoryScopeRequest{
			Id:              id,
			RepositoryScope: repositoryScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveTeamBaseRepositoryScopeByName removes a base repository scope from a team by name.
func (s *teamService) RemoveTeamBaseRepositoryScopeByName(
	ctx context.Context,
	name string,
	organizationName string,
	repositoryScope v1alpha1.RepositoryScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveTeamBaseRepositoryScopeByName(
		ctx,
		&v1alpha1.RemoveTeamBaseRepositoryScopeByNameRequest{
			Name:             name,
			OrganizationName: organizationName,
			RepositoryScope:  repositoryScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// AddTeamRepositoryScope adds a repository scope for a specific repository to a team by ID.
func (s *teamService) AddTeamRepositoryScope(
	ctx context.Context,
	id string,
	repositoryId string,
	repositoryScope v1alpha1.RepositoryScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddTeamRepositoryScope(
		ctx,
		&v1alpha1.AddTeamRepositoryScopeRequest{
			Id:              id,
			RepositoryId:    repositoryId,
			RepositoryScope: repositoryScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// AddTeamRepositoryScopeByName adds a repository scope for a specific repository to a team by name.
func (s *teamService) AddTeamRepositoryScopeByName(
	ctx context.Context,
	name string,
	organizationName string,
	repositoryName string,
	repositoryScope v1alpha1.RepositoryScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddTeamRepositoryScopeByName(
		ctx,
		&v1alpha1.AddTeamRepositoryScopeByNameRequest{
			Name:             name,
			OrganizationName: organizationName,
			RepositoryName:   repositoryName,
			RepositoryScope:  repositoryScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveTeamRepositoryScope removes a repository scope for a specific repository from a team by ID.
func (s *teamService) RemoveTeamRepositoryScope(
	ctx context.Context,
	id string,
	repositoryId string,
	repositoryScope v1alpha1.RepositoryScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveTeamRepositoryScope(
		ctx,
		&v1alpha1.RemoveTeamRepositoryScopeRequest{
			Id:              id,
			RepositoryId:    repositoryId,
			RepositoryScope: repositoryScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveTeamRepositoryScopeByName removes a repository scope for a specific repository from a team by name.
func (s *teamService) RemoveTeamRepositoryScopeByName(
	ctx context.Context,
	name string,
	organizationName string,
	repositoryName string,
	repositoryScope v1alpha1.RepositoryScope,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveTeamRepositoryScopeByName(
		ctx,
		&v1alpha1.RemoveTeamRepositoryScopeByNameRequest{
			Name:             name,
			OrganizationName: organizationName,
			RepositoryName:   repositoryName,
			RepositoryScope:  repositoryScope,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
