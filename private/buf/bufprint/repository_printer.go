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

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
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

func (p *repositoryPrinter) PrintRepository(ctx context.Context, format Format, message *modulev1.Module) error {
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

func (p *repositoryPrinter) PrintRepositories(ctx context.Context, format Format, nextPageToken string, messages ...*modulev1.Module) error {
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

func (p *repositoryPrinter) registryRepositoriesToOutRepositories(ctx context.Context, messages ...*modulev1.Module) ([]outputRepository, error) {
	ownerRefs := slicesext.Map(messages, func(module *modulev1.Module) *ownerv1.OwnerRef {
		return &ownerv1.OwnerRef{
			Value: &ownerv1.OwnerRef_Id{
				Id: module.OwnerId,
			},
		}
	})
	ownerServiceClient := bufapi.NewClientProvider(p.clientConfig).V1OwnerServiceClient(p.address)
	resp, err := ownerServiceClient.GetOwners(
		ctx,
		&connect.Request[ownerv1.GetOwnersRequest]{
			Msg: &ownerv1.GetOwnersRequest{
				OwnerRefs: ownerRefs,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	owners := resp.Msg.GetOwners()
	if len(owners) != len(messages) {
		return nil, syserror.Newf("expected %d owners from response, got %d", len(messages), len(owners))
	}
	outputRepositories := make([]outputRepository, len(messages))
	for i, module := range messages {
		var ownerName string
		owner := owners[i]
		switch {
		case owner.GetUser() != nil:
			ownerName = owner.GetUser().Name
		case owner.GetOrganization() != nil:
			ownerName = owner.GetOrganization().Name
		default:
			return nil, syserror.Newf("owner with id %s is neither a user nor an organization", messages[i].GetOwnerId())
		}
		outputRepositories[i] = outputRepository{
			ID:         module.GetId(),
			Remote:     p.address,
			Owner:      ownerName,
			Name:       module.GetName(),
			CreateTime: module.GetCreateTime().AsTime(),
		}
	}
	return outputRepositories, nil
}

func (p *repositoryPrinter) printRepositoriesText(outputRepositories []outputRepository) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"Full Name",
			"Create Time",
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
