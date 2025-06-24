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
	"sync"
)

// PolicyData presents the raw Policy data read by PolicyKey.
//
// A PolicyData generally represents the data on a Policy read from the BSR API
// or a cache.
//
// Tamper-proofing is done as part of every function.
type PolicyData interface {
	// PolicyKey used to download this PolicyData.
	//
	// The Digest from this PolicyKey is used for tamper-proofing. It will be checked
	// against the actual data downloaded before Data() returns.
	PolicyKey() PolicyKey
	// Config returns the PolicyConfig for the Policy.
	Config() (PolicyConfig, error)

	isPolicyData()
}

// NewPolicyData returns a new PolicyData.
//
// getData is expected to be lazily-loaded function where possible.
func NewPolicyData(
	ctx context.Context,
	policyKey PolicyKey,
	getConfig func() (PolicyConfig, error),
) (PolicyData, error) {
	return newPolicyData(
		ctx,
		policyKey,
		getConfig,
	)
}

// *** PRIVATE ***

type policyData struct {
	policyKey PolicyKey
	getConfig func() (PolicyConfig, error)

	checkDigest func() error
}

func newPolicyData(
	ctx context.Context,
	policyKey PolicyKey,
	getConfig func() (PolicyConfig, error),
) (*policyData, error) {
	policyData := &policyData{
		policyKey: policyKey,
		getConfig: getConfig,
	}
	policyData.checkDigest = sync.OnceValue(func() error {
		policyConfig, err := policyData.getConfig()
		if err != nil {
			return err
		}
		actualDigest, err := getO1Digest(policyConfig)
		if err != nil {
			return err
		}
		expectedDigest, err := policyKey.Digest()
		if err != nil {
			return err
		}
		if !DigestEqual(actualDigest, expectedDigest) {
			return &DigestMismatchError{
				FullName:       policyKey.FullName(),
				CommitID:       policyKey.CommitID(),
				ExpectedDigest: expectedDigest,
				ActualDigest:   actualDigest,
			}
		}
		return nil
	})
	return policyData, nil
}

func (p *policyData) PolicyKey() PolicyKey {
	return p.policyKey
}

func (p *policyData) Config() (PolicyConfig, error) {
	if err := p.checkDigest(); err != nil {
		return nil, err
	}
	return p.getConfig()
}

func (*policyData) isPolicyData() {}
