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

package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	assert.Equal(t, "", GetIncomingHeader(ctx, "foo"))
	assert.Equal(t, "", GetOutgoingHeader(ctx, "foo"))
	assert.NotNil(t, GetIncomingHeaders(ctx))
	assert.Empty(t, GetIncomingHeaders(ctx))
	assert.NotNil(t, GetOutgoingHeaders(ctx))
	assert.Empty(t, GetOutgoingHeaders(ctx))

	ctx = WithIncomingHeader(ctx, "foo", "foo1")
	assert.Equal(t, "foo1", GetIncomingHeader(ctx, "foo"))
	assert.Equal(t, "", GetOutgoingHeader(ctx, "foo"))
	assert.Equal(t, map[string]string{"foo": "foo1"}, GetIncomingHeaders(ctx))
	assert.Empty(t, GetOutgoingHeaders(ctx))

	ctx = WithIncomingHeader(ctx, "Foo", "foo2")
	assert.Equal(t, "foo2", GetIncomingHeader(ctx, "Foo"))
	assert.Equal(t, "foo2", GetIncomingHeader(ctx, "foo"))
	assert.Equal(t, "", GetOutgoingHeader(ctx, "Foo"))
	assert.Equal(t, "", GetOutgoingHeader(ctx, "foo"))
	assert.Equal(t, map[string]string{"foo": "foo2"}, GetIncomingHeaders(ctx))
	assert.Empty(t, GetOutgoingHeaders(ctx))

	ctx = WithOutgoingHeader(ctx, "bar", "bar1")
	assert.Equal(t, "bar1", GetOutgoingHeader(ctx, "bar"))
	assert.Equal(t, map[string]string{"bar": "bar1"}, GetOutgoingHeaders(ctx))

	ctx = WithOutgoingHeader(ctx, "Bar", "bar2")
	assert.Equal(t, "bar2", GetOutgoingHeader(ctx, "Bar"))
	assert.Equal(t, "bar2", GetOutgoingHeader(ctx, "bar"))
	assert.Equal(t, map[string]string{"bar": "bar2"}, GetOutgoingHeaders(ctx))

	headers := map[string]string{
		"foo": "foo3",
		"bar": "bar3",
	}
	headers2 := map[string]string{
		"foo": "foo4",
		"bar": "bar4",
		"baz": "baz4",
	}
	ctx = WithIncomingHeaders(ctx, headers)
	assert.Equal(t, headers, GetIncomingHeaders(ctx))
	ctx = WithIncomingHeaders(ctx, headers2)
	assert.Equal(t, headers2, GetIncomingHeaders(ctx))
	// make sure that the headers map is not modified
	assert.Equal(
		t,
		map[string]string{
			"foo": "foo3",
			"bar": "bar3",
		},
		headers,
	)
	ctx = WithOutgoingHeaders(ctx, headers)
	assert.Equal(t, headers, GetOutgoingHeaders(ctx))
	ctx = WithOutgoingHeaders(ctx, headers2)
	assert.Equal(t, headers2, GetOutgoingHeaders(ctx))
	// make sure that the headers map is not modified
	assert.Equal(
		t,
		map[string]string{
			"foo": "foo3",
			"bar": "bar3",
		},
		headers,
	)
}
