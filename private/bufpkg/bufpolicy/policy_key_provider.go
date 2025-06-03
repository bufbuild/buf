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

package bufpolicy

import (
	"context"
	"fmt"
	"io/fs"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
)

var (
	// NopPolicyKeyProvider is a no-op PolicyKeyProvider.
	NopPolicyKeyProvider PolicyKeyProvider = nopPolicyKeyProvider{}
)

// PolicyKeyProvider provides PolicyKeys for bufparse.Refs.
type PolicyKeyProvider interface {
	// GetPolicyKeysForPolicyRefs gets the PolicyKets for the given PolicyRefs.
	//
	// Returned PolicyKeys will be in the same order as the input PolicyRefs.
	//
	// The input PolicyRefs are expected to be unique by FullName. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the PolicyKeys returned will match the length of the Refs.
	// If there is an error, no PolicyKeys will be returned.
	// If any PolicyRef is not found, an error with fs.ErrNotExist will be returned.
	GetPolicyKeysForPolicyRefs(context.Context, []bufparse.Ref, DigestType) ([]PolicyKey, error)
}

// NewStaticPolicyKeyProvider returns a new PolicyKeyProvider for a static set of PolicyKeys.
//
// The set of PolicyKeys must be unique by FullName. If there are duplicates,
// an error will be returned.
//
// When resolving Refs, the Ref will be matched to the PolicyKey by FullName.
// If the Ref is not found in the set of provided keys, an fs.ErrNotExist will be returned.
func NewStaticPolicyKeyProvider(policyKeys []PolicyKey) (PolicyKeyProvider, error) {
	return newStaticPolicyKeyProvider(policyKeys)
}

// *** PRIVATE ***

type nopPolicyKeyProvider struct{}

func (nopPolicyKeyProvider) GetPolicyKeysForPolicyRefs(
	context.Context,
	[]bufparse.Ref,
	DigestType,
) ([]PolicyKey, error) {
	return nil, fs.ErrNotExist
}

type staticPolicyKeyProvider struct {
	policyKeysByFullName map[string]PolicyKey
}

func newStaticPolicyKeyProvider(policyKeys []PolicyKey) (*staticPolicyKeyProvider, error) {
	var policyKeysByFullName map[string]PolicyKey
	if len(policyKeys) > 0 {
		var err error
		policyKeysByFullName, err = xslices.ToUniqueValuesMap(policyKeys, func(policyKey PolicyKey) string {
			return policyKey.FullName().String()
		})
		if err != nil {
			return nil, err
		}
	}
	return &staticPolicyKeyProvider{
		policyKeysByFullName: policyKeysByFullName,
	}, nil
}

func (s staticPolicyKeyProvider) GetPolicyKeysForPolicyRefs(
	_ context.Context,
	refs []bufparse.Ref,
	digestType DigestType,
) ([]PolicyKey, error) {
	policyKeys := make([]PolicyKey, len(refs))
	for i, ref := range refs {
		// Only the FullName is used to match the PolicyKey. The Ref is not
		// validated to match the PolicyKey as there is not enough information
		// to do so.
		policyKey, ok := s.policyKeysByFullName[ref.FullName().String()]
		if !ok {
			return nil, fs.ErrNotExist
		}
		digest, err := policyKey.Digest()
		if err != nil {
			return nil, err
		}
		if digest.Type() != digestType {
			return nil, fmt.Errorf("expected DigestType %v, got %v", digestType, digest.Type())
		}
		policyKeys[i] = policyKey
	}
	return policyKeys, nil
}
