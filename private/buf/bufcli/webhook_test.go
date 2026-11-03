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
	"testing"

	webhookv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/webhook/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWebhookScope(t *testing.T) {
	t.Parallel()

	registry, scopeRef, err := ParseWebhookScope("buf.build")
	require.NoError(t, err)
	assert.Equal(t, "buf.build", registry)
	assert.NotNil(t, scopeRef.GetInstance())

	registry, scopeRef, err = ParseWebhookScope("buf.build/acme")
	require.NoError(t, err)
	assert.Equal(t, "buf.build", registry)
	assert.Equal(t, "acme", scopeRef.GetOrganizationRef().GetName())

	registry, scopeRef, err = ParseWebhookScope("buf.build/acme/petapis")
	require.NoError(t, err)
	assert.Equal(t, "buf.build", registry)
	assert.Equal(t, "acme", scopeRef.GetModuleRef().GetName().GetOwner())
	assert.Equal(t, "petapis", scopeRef.GetModuleRef().GetName().GetModule())

	for _, invalid := range []string{
		"",
		"buf.build/",
		"/acme",
		"buf.build/acme/",
		"buf.build/acme/petapis/extra",
		"not a hostname",
	} {
		_, _, err := ParseWebhookScope(invalid)
		assert.Error(t, err, "input %q", invalid)
	}
}

func TestNewWebhookRef(t *testing.T) {
	t.Parallel()
	_, scopeRef, err := ParseWebhookScope("buf.build/acme")
	require.NoError(t, err)

	webhookRef, err := NewWebhookRef(scopeRef, "ci", "")
	require.NoError(t, err)
	assert.Equal(t, "ci", webhookRef.GetName().GetName())
	assert.Equal(t, scopeRef, webhookRef.GetName().GetScopeRef())

	webhookRef, err = NewWebhookRef(scopeRef, "", "abc")
	require.NoError(t, err)
	assert.Equal(t, "abc", webhookRef.GetId())

	_, err = NewWebhookRef(scopeRef, "ci", "abc")
	assert.Error(t, err)
	_, err = NewWebhookRef(scopeRef, "", "")
	assert.Error(t, err)
}

func TestParseWebhookEventType(t *testing.T) {
	t.Parallel()

	eventType, err := ParseWebhookEventType("EVENT_TYPE_MODULE_COMMITS_PUSHED")
	require.NoError(t, err)
	assert.Equal(t, webhookv1.EventType_EVENT_TYPE_MODULE_COMMITS_PUSHED, eventType)

	for _, invalid := range []string{
		"",
		"EVENT_TYPE_UNSPECIFIED",
		"EVENT_TYPE_PING",
		"MODULE_COMMITS_PUSHED",
		"event_type_module_commits_pushed",
	} {
		_, err := ParseWebhookEventType(invalid)
		assert.Error(t, err, "input %q", invalid)
	}
}
