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
		fqn      []string
		prefix   []string
		expected bool
	}{
		{
			name:     "exact match",
			fqn:      []string{"foo", "bar", "baz"},
			prefix:   []string{"foo", "bar", "baz"},
			expected: true,
		},
		{
			name:     "prefix match",
			fqn:      []string{"foo", "bar", "baz"},
			prefix:   []string{"foo", "bar"},
			expected: true,
		},
		{
			name:     "single component prefix",
			fqn:      []string{"foo", "bar", "baz"},
			prefix:   []string{"foo"},
			expected: true,
		},
		{
			name:     "empty prefix matches all",
			fqn:      []string{"foo", "bar"},
			prefix:   []string{},
			expected: true,
		},
		{
			name:     "prefix longer than fqn",
			fqn:      []string{"foo", "bar"},
			prefix:   []string{"foo", "bar", "baz"},
			expected: false,
		},
		{
			name:     "partial component mismatch - foo.bar.b does not match foo.bar.baz",
			fqn:      []string{"foo", "bar", "baz"},
			prefix:   []string{"foo", "bar", "b"},
			expected: false,
		},
		{
			name:     "different path",
			fqn:      []string{"foo", "bar", "baz"},
			prefix:   []string{"foo", "qux"},
			expected: false,
		},
		{
			name:     "empty fqn",
			fqn:      []string{},
			prefix:   []string{"foo"},
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

func TestFQNMatchesExact(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		fqn      []string
		prefix   []string
		expected bool
	}{
		{
			name:     "exact match",
			fqn:      []string{"foo", "bar", "baz"},
			prefix:   []string{"foo", "bar", "baz"},
			expected: true,
		},
		{
			name:     "prefix only - not exact",
			fqn:      []string{"foo", "bar", "baz"},
			prefix:   []string{"foo", "bar"},
			expected: false,
		},
		{
			name:     "longer prefix - not exact",
			fqn:      []string{"foo", "bar"},
			prefix:   []string{"foo", "bar", "baz"},
			expected: false,
		},
		{
			name:     "both empty",
			fqn:      []string{},
			prefix:   []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := fqnMatchesExact(tt.fqn, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeprecationCheckerShouldDeprecate(t *testing.T) {
	t.Parallel()
	checker := newDeprecationChecker([]string{"foo.bar", "baz.qux"})

	tests := []struct {
		name     string
		fqn      []string
		expected bool
	}{
		{
			name:     "matches first prefix",
			fqn:      []string{"foo", "bar", "baz"},
			expected: true,
		},
		{
			name:     "matches second prefix",
			fqn:      []string{"baz", "qux", "quux"},
			expected: true,
		},
		{
			name:     "exact match on prefix",
			fqn:      []string{"foo", "bar"},
			expected: true,
		},
		{
			name:     "no match",
			fqn:      []string{"other", "package"},
			expected: false,
		},
		{
			name:     "partial component no match",
			fqn:      []string{"foo", "bart"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := checker.shouldDeprecate(tt.fqn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeprecationCheckerShouldDeprecateExact(t *testing.T) {
	t.Parallel()
	checker := newDeprecationChecker([]string{"foo.bar.baz.SomeMessage.some_field"})

	tests := []struct {
		name     string
		fqn      []string
		expected bool
	}{
		{
			name:     "exact match",
			fqn:      []string{"foo", "bar", "baz", "SomeMessage", "some_field"},
			expected: true,
		},
		{
			name:     "prefix only - not exact",
			fqn:      []string{"foo", "bar", "baz", "SomeMessage"},
			expected: false,
		},
		{
			name:     "longer than prefix",
			fqn:      []string{"foo", "bar", "baz", "SomeMessage", "some_field", "extra"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := checker.shouldDeprecateExact(tt.fqn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeprecationCheckerEmptyPrefixes(t *testing.T) {
	t.Parallel()
	checker := newDeprecationChecker([]string{})

	// Empty prefixes should not match anything
	assert.True(t, checker.isEmpty())
	assert.False(t, checker.shouldDeprecate([]string{"foo", "bar"}))
	assert.False(t, checker.shouldDeprecateExact([]string{"foo", "bar"}))
}

func TestDeprecationCheckerEmptyStringsFiltered(t *testing.T) {
	t.Parallel()
	checker := newDeprecationChecker([]string{"", "foo.bar", ""})

	// Empty strings should be filtered out
	assert.False(t, checker.isEmpty())
	assert.True(t, checker.shouldDeprecate([]string{"foo", "bar", "baz"}))
}
