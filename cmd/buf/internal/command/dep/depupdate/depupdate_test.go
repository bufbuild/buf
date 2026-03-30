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

package depupdate

import (
	"context"
	"fmt"
	"testing"

	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveModuleRefsWithFallback_OverriddenAndNonOverridden tests the scenario
// where dependency A is overridden with a branch label (which doesn't exist on BSR)
// and dependency B is not overridden. Both should resolve successfully: A via fallback
// to its original label, B directly.
func TestResolveModuleRefsWithFallback_OverriddenAndNonOverridden(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	logger := slogtestext.NewLogger(t)

	// A's original ref has a hard-coded label "v1.2.0".
	refAOriginal := mustNewRef(t, "buf.build", "acme", "weather", "v1.2.0")
	// A is overridden to use branch label "feature/test".
	refABranch := mustNewRef(t, "buf.build", "acme", "weather", "feature/test")
	// B is not overridden, no label (uses default).
	refB := mustNewRef(t, "buf.build", "acme", "petapis", "")

	moduleKeyA := mustNewModuleKey(t, "buf.build", "acme", "weather")
	moduleKeyB := mustNewModuleKey(t, "buf.build", "acme", "petapis")

	originalRefs := map[string]bufparse.Ref{
		"buf.build/acme/weather": refAOriginal,
	}

	// Mock resolver: "feature/test" label doesn't exist for A, everything else works.
	mockResolver := func(
		_ context.Context,
		_ appext.Container,
		refs []bufparse.Ref,
		_ bufmodule.DigestType,
	) ([]bufmodule.ModuleKey, error) {
		// Batch call with both refs: fail because A's branch label doesn't exist.
		if len(refs) > 1 {
			for _, ref := range refs {
				if ref.Ref() == "feature/test" {
					return nil, fmt.Errorf("label %q not found for module %s", ref.Ref(), ref.FullName())
				}
			}
		}
		// Single ref calls.
		ref := refs[0]
		switch {
		case ref.FullName().String() == "buf.build/acme/weather" && ref.Ref() == "feature/test":
			return nil, fmt.Errorf("label %q not found for module %s", ref.Ref(), ref.FullName())
		case ref.FullName().String() == "buf.build/acme/weather" && ref.Ref() == "v1.2.0":
			return []bufmodule.ModuleKey{moduleKeyA}, nil
		case ref.FullName().String() == "buf.build/acme/petapis":
			return []bufmodule.ModuleKey{moduleKeyB}, nil
		default:
			return nil, fmt.Errorf("unexpected ref: %s", ref)
		}
	}

	keys, err := doResolveModuleRefsWithFallback(
		ctx,
		nil, // container not used by mock
		logger,
		[]bufparse.Ref{refABranch, refB},
		originalRefs,
		bufmodule.DigestTypeB5,
		mockResolver,
	)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Equal(t, "buf.build/acme/weather", keys[0].FullName().String())
	assert.Equal(t, "buf.build/acme/petapis", keys[1].FullName().String())
}

// TestResolveModuleRefsWithFallback_BatchSucceeds tests that when the batch
// call succeeds, no fallback is needed.
func TestResolveModuleRefsWithFallback_BatchSucceeds(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	logger := slogtestext.NewLogger(t)

	refA := mustNewRef(t, "buf.build", "acme", "weather", "feature/test")
	refB := mustNewRef(t, "buf.build", "acme", "petapis", "")

	moduleKeyA := mustNewModuleKey(t, "buf.build", "acme", "weather")
	moduleKeyB := mustNewModuleKey(t, "buf.build", "acme", "petapis")

	originalRefs := map[string]bufparse.Ref{
		"buf.build/acme/weather": mustNewRef(t, "buf.build", "acme", "weather", "v1.2.0"),
	}

	callCount := 0
	mockResolver := func(
		_ context.Context,
		_ appext.Container,
		refs []bufparse.Ref,
		_ bufmodule.DigestType,
	) ([]bufmodule.ModuleKey, error) {
		callCount++
		return []bufmodule.ModuleKey{moduleKeyA, moduleKeyB}, nil
	}

	keys, err := doResolveModuleRefsWithFallback(
		ctx,
		nil,
		logger,
		[]bufparse.Ref{refA, refB},
		originalRefs,
		bufmodule.DigestTypeB5,
		mockResolver,
	)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Equal(t, 1, callCount, "should only call resolver once when batch succeeds")
}

// TestResolveModuleRefsWithFallback_NonOverriddenFails tests that when a
// non-overridden ref fails, the error is returned immediately.
func TestResolveModuleRefsWithFallback_NonOverriddenFails(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	logger := slogtestext.NewLogger(t)

	refA := mustNewRef(t, "buf.build", "acme", "weather", "feature/test")
	refB := mustNewRef(t, "buf.build", "acme", "petapis", "")

	originalRefs := map[string]bufparse.Ref{
		"buf.build/acme/weather": mustNewRef(t, "buf.build", "acme", "weather", "v1.2.0"),
	}

	mockResolver := func(
		_ context.Context,
		_ appext.Container,
		refs []bufparse.Ref,
		_ bufmodule.DigestType,
	) ([]bufmodule.ModuleKey, error) {
		// Batch always fails.
		if len(refs) > 1 {
			return nil, fmt.Errorf("batch failed")
		}
		ref := refs[0]
		if ref.FullName().String() == "buf.build/acme/petapis" {
			return nil, fmt.Errorf("module not found: %s", ref.FullName())
		}
		// A with branch label fails too.
		if ref.Ref() == "feature/test" {
			return nil, fmt.Errorf("label not found")
		}
		return []bufmodule.ModuleKey{mustNewModuleKey(t, ref.FullName().Registry(), ref.FullName().Owner(), ref.FullName().Name())}, nil
	}

	_, err := doResolveModuleRefsWithFallback(
		ctx,
		nil,
		logger,
		[]bufparse.Ref{refA, refB},
		originalRefs,
		bufmodule.DigestTypeB5,
		mockResolver,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module not found: buf.build/acme/petapis")
}

// TestResolveModuleRefsWithFallback_NoOverrides tests that when no overrides
// were applied, a batch failure is returned directly without per-ref fallback.
func TestResolveModuleRefsWithFallback_NoOverrides(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	logger := slogtestext.NewLogger(t)

	refA := mustNewRef(t, "buf.build", "acme", "weather", "")

	callCount := 0
	mockResolver := func(
		_ context.Context,
		_ appext.Container,
		_ []bufparse.Ref,
		_ bufmodule.DigestType,
	) ([]bufmodule.ModuleKey, error) {
		callCount++
		return nil, fmt.Errorf("not found")
	}

	_, err := doResolveModuleRefsWithFallback(
		ctx,
		nil,
		logger,
		[]bufparse.Ref{refA},
		nil, // no overrides
		bufmodule.DigestTypeB5,
		mockResolver,
	)
	require.Error(t, err)
	assert.Equal(t, 1, callCount, "should only call resolver once when no overrides")
}

func mustNewRef(t *testing.T, registry, owner, name, ref string) bufparse.Ref {
	t.Helper()
	moduleRef, err := bufparse.NewRef(registry, owner, name, ref)
	require.NoError(t, err)
	return moduleRef
}

func mustNewModuleKey(t *testing.T, registry, owner, name string) bufmodule.ModuleKey {
	t.Helper()
	fullName, err := bufparse.NewFullName(registry, owner, name)
	require.NoError(t, err)
	moduleKey, err := bufmodule.NewModuleKey(
		fullName,
		uuid.New(),
		func() (bufmodule.Digest, error) {
			return nil, fmt.Errorf("digest not implemented in test")
		},
	)
	require.NoError(t, err)
	return moduleKey
}
