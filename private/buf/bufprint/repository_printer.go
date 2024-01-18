// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufprint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/connectclient"
)

type repositoryPrinter struct {
	clientConfig *connectclient.Config
	address      string
	writer       io.Writer
}

func newRepositoryPrinter(
	clientConfig *connectclient.Config,
	address string,
	writer io.Writer,
) *repositoryPrinter {
	return &repositoryPrinter{
		clientConfig: clientConfig,
		address:      address,
		writer:       writer,
	}
}

func (p *repositoryPrinter) PrintRepository(ctx context.Context, format Format, message *registryv1alpha1.Repository) error {
	outputRepositories, err := p.registryRepositoriesToOutRepositories(ctx, message)
	if err != nil {
		return err
	}
	if len(outputRepositories) != 1 {
		return fmt.Errorf("error converting repositories: expected 1 got %d", len(outputRepositories))
	}
	switch format {
	case FormatText:
		return p.printRepositoriesText(outputRepositories)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(outputRepositories[0])
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryPrinter) PrintRepositories(ctx context.Context, format Format, nextPageToken string, messages ...*registryv1alpha1.Repository) error {
	if len(messages) == 0 {
		return nil
	}
	outputRepositories, err := p.registryRepositoriesToOutRepositories(ctx, messages...)
	if err != nil {
		return err
	}
	switch format {
	case FormatText:
		return p.printRepositoriesText(outputRepositories)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputRepositories,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryPrinter) registryRepositoriesToOutRepositories(ctx context.Context, messages ...*registryv1alpha1.Repository) ([]outputRepository, error) {
	var outputRepositories []outputRepository
	for _, repository := range messages {
		var ownerName string
		switch owner := repository.Owner.(type) {
		case *registryv1alpha1.Repository_OrganizationId:
			organizationService := connectclient.Make(p.clientConfig, p.address, registryv1alpha1connect.NewOrganizationServiceClient)
			resp, err := organizationService.GetOrganization(
				ctx,
				connect.NewRequest(&registryv1alpha1.GetOrganizationRequest{
					Id: owner.OrganizationId,
				}),
			)
			if err != nil {
				return nil, err
			}
			ownerName = resp.Msg.Organization.Name
		case *registryv1alpha1.Repository_UserId:
			userService := connectclient.Make(p.clientConfig, p.address, registryv1alpha1connect.NewUserServiceClient)
			resp, err := userService.GetUser(
				ctx,
				connect.NewRequest(&registryv1alpha1.GetUserRequest{
					Id: owner.UserId,
				}),
			)
			if err != nil {
				return nil, err
			}
			ownerName = resp.Msg.User.Username
		default:
			return nil, fmt.Errorf("unknown owner: %T", owner)
		}
		outputRepository := outputRepository{
			ID:         repository.Id,
			Remote:     p.address,
			Owner:      ownerName,
			Name:       repository.Name,
			CreateTime: repository.CreateTime.AsTime(),
		}
		outputRepositories = append(outputRepositories, outputRepository)
	}
	return outputRepositories, nil
}

func (p *repositoryPrinter) printRepositoriesText(outputRepositories []outputRepository) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"Full name",
			"Created",
		},
		func(tabWriter TabWriter) error {
			for _, outputRepository := range outputRepositories {
				if err := tabWriter.Write(
					outputRepository.Remote+"/"+outputRepository.Owner+"/"+outputRepository.Name,
					outputRepository.CreateTime.Format(time.RFC3339),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRepository struct {
	ID         string    `json:"id,omitempty"`
	Remote     string    `json:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty"`
	Name       string    `json:"name,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
}
