// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufformat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFQNMatchesPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		fqn      string
		prefix   string
		expected bool
	}{
		{
			name:     "exact match",
			fqn:      "foo.bar.baz",
			prefix:   "foo.bar.baz",
			expected: true,
		},
		{
			name:     "prefix match",
			fqn:      "foo.bar.baz",
			prefix:   "foo.bar",
			expected: true,
		},
		{
			name:     "single component prefix",
			fqn:      "foo.bar.baz",
			prefix:   "foo",
			expected: true,
		},
		{
			name:     "empty prefix matches all",
			fqn:      "foo.bar",
			prefix:   "",
			expected: true,
		},
		{
			name:     "prefix longer than fqn",
			fqn:      "foo.bar",
			prefix:   "foo.bar.baz",
			expected: false,
		},
		{
			name:     "partial component mismatch - foo.bar.b does not match foo.bar.baz",
			fqn:      "foo.bar.baz",
			prefix:   "foo.bar.b",
			expected: false,
		},
		{
			name:     "different path",
			fqn:      "foo.bar.baz",
			prefix:   "foo.qux",
			expected: false,
		},
		{
			name:     "empty fqn",
			fqn:      "",
			prefix:   "foo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := fqnMatchesPrefix(tt.fqn, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFullNameMatcherMatchesPrefix(t *testing.T) {
	t.Parallel()
	matcher := newFullNameMatcher("foo.bar", "baz.qux")

	tests := []struct {
		name     string
		fqn      string
		expected bool
	}{
		{
			name:     "matches first prefix",
			fqn:      "foo.bar.baz",
			expected: true,
		},
		{
			name:     "matches second prefix",
			fqn:      "baz.qux.quux",
			expected: true,
		},
		{
			name:     "exact match on prefix",
			fqn:      "foo.bar",
			expected: true,
		},
		{
			name:     "no match",
			fqn:      "other.package",
			expected: false,
		},
		{
			name:     "partial component no match",
			fqn:      "foo.bart",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := matcher.matchesPrefix(tt.fqn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFullNameMatcherMatchesExact(t *testing.T) {
	t.Parallel()
	matcher := newFullNameMatcher("foo.bar.baz.SomeMessage.some_field")

	tests := []struct {
		name     string
		fqn      string
		expected bool
	}{
		{
			name:     "exact match",
			fqn:      "foo.bar.baz.SomeMessage.some_field",
			expected: true,
		},
		{
			name:     "prefix only - not exact",
			fqn:      "foo.bar.baz.SomeMessage",
			expected: false,
		},
		{
			name:     "longer than prefix",
			fqn:      "foo.bar.baz.SomeMessage.some_field.extra",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := matcher.matchesExact(tt.fqn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFullNameMatcherEmptyPrefixes(t *testing.T) {
	t.Parallel()
	matcher := newFullNameMatcher()

	// Empty prefixes should not match anything
	assert.False(t, matcher.matchesPrefix("foo.bar"))
	assert.False(t, matcher.matchesExact("foo.bar"))
}
