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

package bufcli

import (
	"errors"
	"fmt"
	"strings"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	webhookv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/webhook/v1"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/netext"
)

// ParseWebhookScope parses a webhook scope from a string in the form "registry",
// "registry/organization", or "registry/organization/module".
//
// It returns the registry and a WebhookScopeRef for the instance, organization,
// or module respectively.
func ParseWebhookScope(webhookScopeString string) (string, *webhookv1.WebhookScopeRef, error) {
	switch strings.Count(webhookScopeString, "/") {
	case 0:
		registry := strings.TrimSpace(webhookScopeString)
		if registry == "" {
			return "", nil, errors.New("registry is empty")
		}
		if _, err := netext.ValidateHostname(registry); err != nil {
			return "", nil, fmt.Errorf("registry %q is not a valid hostname: %w", registry, err)
		}
		return registry, &webhookv1.WebhookScopeRef{
			Value: &webhookv1.WebhookScopeRef_Instance_{
				Instance: &webhookv1.WebhookScopeRef_Instance{},
			},
		}, nil
	case 1:
		moduleOwner, err := ParseModuleOwner(webhookScopeString)
		if err != nil {
			return "", nil, err
		}
		return moduleOwner.Registry(), &webhookv1.WebhookScopeRef{
			Value: &webhookv1.WebhookScopeRef_OrganizationRef{
				OrganizationRef: &ownerv1.OrganizationRef{
					Value: &ownerv1.OrganizationRef_Name{
						Name: moduleOwner.Owner(),
					},
				},
			},
		}, nil
	case 2:
		moduleFullName, err := bufparse.ParseFullName(webhookScopeString)
		if err != nil {
			return "", nil, err
		}
		return moduleFullName.Registry(), &webhookv1.WebhookScopeRef{
			Value: &webhookv1.WebhookScopeRef_ModuleRef{
				ModuleRef: &modulev1.ModuleRef{
					Value: &modulev1.ModuleRef_Name_{
						Name: &modulev1.ModuleRef_Name{
							Owner:  moduleFullName.Owner(),
							Module: moduleFullName.Name(),
						},
					},
				},
			},
		}, nil
	default:
		return "", nil, fmt.Errorf(
			"invalid webhook scope %q: must be in the form registry, registry/organization, or registry/organization/module",
			webhookScopeString,
		)
	}
}

// NewWebhookRef returns a WebhookRef for either a webhook name within the
// scope, or a webhook id. Exactly one of webhookName and webhookID must be set.
func NewWebhookRef(
	webhookScopeRef *webhookv1.WebhookScopeRef,
	webhookName string,
	webhookID string,
) (*webhookv1.WebhookRef, error) {
	switch {
	case webhookName != "" && webhookID != "":
		return nil, errors.New("only one of webhook name or id may be set")
	case webhookName != "":
		return &webhookv1.WebhookRef{
			Value: &webhookv1.WebhookRef_Name_{
				Name: &webhookv1.WebhookRef_Name{
					ScopeRef: webhookScopeRef,
					Name:     webhookName,
				},
			},
		}, nil
	case webhookID != "":
		return &webhookv1.WebhookRef{
			Value: &webhookv1.WebhookRef_Id{
				Id: webhookID,
			},
		}, nil
	default:
		return nil, errors.New("one of webhook name or id must be set")
	}
}

// ParseWebhookEventType parses a webhook event type from its enum value name,
// such as "EVENT_TYPE_MODULE_COMMITS_PUSHED".
//
// Only event types that a webhook can subscribe to are accepted.
func ParseWebhookEventType(webhookEventTypeString string) (webhookv1.EventType, error) {
	value, ok := webhookv1.EventType_value[webhookEventTypeString]
	if !ok {
		return 0, fmt.Errorf(
			"unknown webhook event type %q: must be one of %s",
			webhookEventTypeString,
			strings.Join(WebhookEventTypeStrings(), ", "),
		)
	}
	webhookEventType := webhookv1.EventType(value)
	if !isSubscribableWebhookEventType(webhookEventType) {
		return 0, fmt.Errorf(
			"webhook event type %q cannot be subscribed to: must be one of %s",
			webhookEventTypeString,
			strings.Join(WebhookEventTypeStrings(), ", "),
		)
	}
	return webhookEventType, nil
}

// WebhookEventTypeStrings returns the enum value names of the webhook event
// types that a webhook can subscribe to.
func WebhookEventTypeStrings() []string {
	enumValueDescriptors := webhookv1.EventType(0).Descriptor().Values()
	var webhookEventTypeStrings []string
	for i := range enumValueDescriptors.Len() {
		enumValueDescriptor := enumValueDescriptors.Get(i)
		if !isSubscribableWebhookEventType(webhookv1.EventType(enumValueDescriptor.Number())) {
			continue
		}
		webhookEventTypeStrings = append(webhookEventTypeStrings, string(enumValueDescriptor.Name()))
	}
	return webhookEventTypeStrings
}

// *** PRIVATE ***

func isSubscribableWebhookEventType(webhookEventType webhookv1.EventType) bool {
	switch webhookEventType {
	case webhookv1.EventType_EVENT_TYPE_UNSPECIFIED, webhookv1.EventType_EVENT_TYPE_PING:
		return false
	default:
		return true
	}
}
