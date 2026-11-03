// Copyright 2020-2026 Buf Technologies, Inc.
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

// Package internal contains shared functionality for the "buf registry webhook" commands.
package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	webhookv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/webhook/v1"
	"buf.build/go/app"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiowner"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// ScopeArgUsage is the usage of the scope argument shared by all webhook commands.
	ScopeArgUsage = "<remote[/organization[/module]]>"
	// ScopeArgLong describes the scope argument shared by all webhook commands.
	ScopeArgLong = `The first argument is the scope of the webhook, one of:

  <remote>                         the BSR instance, such as buf.build
  <remote>/<organization>          an organization, such as buf.build/acme
  <remote>/<organization>/<module> a module, such as buf.build/acme/petapis`
)

// EventFlagUsage returns the usage of an event type flag.
func EventFlagUsage(prefix string) string {
	return fmt.Sprintf(
		"%s. May be repeated. Must be one of %s",
		prefix,
		strings.Join(bufcli.WebhookEventTypeStrings(), ", "),
	)
}

// ParseWebhookEventTypes parses webhook event types from their enum value names.
func ParseWebhookEventTypes(webhookEventTypeStrings []string) ([]webhookv1.EventType, error) {
	webhookEventTypes := make([]webhookv1.EventType, 0, len(webhookEventTypeStrings))
	for _, webhookEventTypeString := range webhookEventTypeStrings {
		webhookEventType, err := bufcli.ParseWebhookEventType(webhookEventTypeString)
		if err != nil {
			return nil, err
		}
		webhookEventTypes = append(webhookEventTypes, webhookEventType)
	}
	return webhookEventTypes, nil
}

// ReadSigningSecretFile reads a signing secret from a file path, or from stdin
// if the path is "-". Trailing newlines are removed.
func ReadSigningSecretFile(container app.StdinContainer, path string) ([]byte, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(container.Stdin())
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read signing secret: %w", err)
	}
	signingSecret := bytes.TrimRight(data, "\r\n")
	if len(signingSecret) == 0 {
		return nil, fmt.Errorf("signing secret from %q is empty", path)
	}
	return signingSecret, nil
}

// NewWebhookEntities returns entities to print for the webhooks.
//
// Webhooks only reference their scope by id, so this looks up the names of
// the organizations and modules the webhooks are scoped to.
func NewWebhookEntities(
	ctx context.Context,
	clientConfig *connectclient.Config,
	registry string,
	webhooks []*webhookv1.Webhook,
) ([]bufprint.Entity, error) {
	var organizationIDs []string
	var moduleIDs []string
	for _, webhook := range webhooks {
		switch value := webhook.GetScope().GetValue().(type) {
		case *webhookv1.WebhookScope_Instance_:
		case *webhookv1.WebhookScope_Organization_:
			organizationIDs = appendUnique(organizationIDs, value.Organization.GetOrganizationId())
		case *webhookv1.WebhookScope_Module_:
			moduleIDs = appendUnique(moduleIDs, value.Module.GetModuleId())
		default:
			return nil, syserror.Newf("unknown webhook scope: %T", value)
		}
	}
	ownerClientProvider := bufregistryapiowner.NewClientProvider(clientConfig)
	organizationIDToName, err := getOrganizationIDToName(ctx, ownerClientProvider, registry, organizationIDs)
	if err != nil {
		return nil, err
	}
	moduleIDToFullName, err := getModuleIDToFullName(
		ctx,
		bufregistryapimodule.NewClientProvider(clientConfig),
		ownerClientProvider,
		registry,
		moduleIDs,
	)
	if err != nil {
		return nil, err
	}
	entities := make([]bufprint.Entity, 0, len(webhooks))
	for _, webhook := range webhooks {
		var scope string
		switch value := webhook.GetScope().GetValue().(type) {
		case *webhookv1.WebhookScope_Instance_:
			scope = registry
		case *webhookv1.WebhookScope_Organization_:
			scope = registry + "/" + organizationIDToName[value.Organization.GetOrganizationId()]
		case *webhookv1.WebhookScope_Module_:
			scope = registry + "/" + moduleIDToFullName[value.Module.GetModuleId()]
		}
		entities = append(entities, bufprint.NewWebhookEntity(webhook, scope))
	}
	return entities, nil
}

// *** PRIVATE ***

func getOrganizationIDToName(
	ctx context.Context,
	ownerClientProvider bufregistryapiowner.ClientProvider,
	registry string,
	organizationIDs []string,
) (map[string]string, error) {
	organizationIDToName := make(map[string]string, len(organizationIDs))
	if len(organizationIDs) == 0 {
		return organizationIDToName, nil
	}
	organizationRefs := make([]*ownerv1.OrganizationRef, 0, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		organizationRefs = append(organizationRefs, &ownerv1.OrganizationRef{
			Value: &ownerv1.OrganizationRef_Id{Id: organizationID},
		})
	}
	resp, err := ownerClientProvider.V1OrganizationServiceClient(registry).GetOrganizations(
		ctx,
		&ownerv1.GetOrganizationsRequest{
			OrganizationRefs: organizationRefs,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, organization := range resp.GetOrganizations() {
		organizationIDToName[organization.GetId()] = organization.GetName()
	}
	return organizationIDToName, nil
}

// getModuleIDToFullName returns a map from module id to "owner/module".
func getModuleIDToFullName(
	ctx context.Context,
	moduleClientProvider bufregistryapimodule.ClientProvider,
	ownerClientProvider bufregistryapiowner.ClientProvider,
	registry string,
	moduleIDs []string,
) (map[string]string, error) {
	moduleIDToFullName := make(map[string]string, len(moduleIDs))
	if len(moduleIDs) == 0 {
		return moduleIDToFullName, nil
	}
	moduleRefs := make([]*modulev1.ModuleRef, 0, len(moduleIDs))
	for _, moduleID := range moduleIDs {
		moduleRefs = append(moduleRefs, &modulev1.ModuleRef{
			Value: &modulev1.ModuleRef_Id{Id: moduleID},
		})
	}
	modulesResp, err := moduleClientProvider.V1ModuleServiceClient(registry).GetModules(
		ctx,
		&modulev1.GetModulesRequest{
			ModuleRefs: moduleRefs,
		},
	)
	if err != nil {
		return nil, err
	}
	modules := modulesResp.GetModules()
	var ownerIDs []string
	for _, module := range modules {
		ownerIDs = appendUnique(ownerIDs, module.GetOwnerId())
	}
	ownerRefs := make([]*ownerv1.OwnerRef, 0, len(ownerIDs))
	for _, ownerID := range ownerIDs {
		ownerRefs = append(ownerRefs, &ownerv1.OwnerRef{
			Value: &ownerv1.OwnerRef_Id{Id: ownerID},
		})
	}
	ownersResp, err := ownerClientProvider.V1OwnerServiceClient(registry).GetOwners(
		ctx,
		&ownerv1.GetOwnersRequest{
			OwnerRefs: ownerRefs,
		},
	)
	if err != nil {
		return nil, err
	}
	ownerIDToName := make(map[string]string, len(ownerIDs))
	for _, owner := range ownersResp.GetOwners() {
		switch value := owner.GetValue().(type) {
		case *ownerv1.Owner_User:
			ownerIDToName[value.User.GetId()] = value.User.GetName()
		case *ownerv1.Owner_Organization:
			ownerIDToName[value.Organization.GetId()] = value.Organization.GetName()
		default:
			return nil, syserror.Newf("unknown owner: %T", value)
		}
	}
	for _, module := range modules {
		moduleIDToFullName[module.GetId()] = ownerIDToName[module.GetOwnerId()] + "/" + module.GetName()
	}
	return moduleIDToFullName, nil
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}
