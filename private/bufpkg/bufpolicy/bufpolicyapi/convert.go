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

package bufpolicyapi

import (
	"fmt"

	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
)

var (
	v1beta1ProtoDigestTypeToDigestType = map[policyv1beta1.DigestType]bufpolicy.DigestType{
		policyv1beta1.DigestType_DIGEST_TYPE_P1: bufpolicy.DigestTypeP1,
	}
)

// V1Beta1ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func V1Beta1ProtoToDigest(protoDigest *policyv1beta1.Digest) (bufpolicy.Digest, error) {
	digestType, err := v1beta1ProtoToDigestType(protoDigest.Type)
	if err != nil {
		return nil, err
	}
	bufcasDigest, err := bufcas.NewDigest(protoDigest.Value)
	if err != nil {
		return nil, err
	}
	return bufpolicy.NewDigest(digestType, bufcasDigest)
}

// *** PRIVATE ***

func policyVisibilityToV1Beta1Proto(policyVisibility bufpolicy.PolicyVisibility) (policyv1beta1.PolicyVisibility, error) {
	switch policyVisibility {
	case bufpolicy.PolicyVisibilityPublic:
		return policyv1beta1.PolicyVisibility_POLICY_VISIBILITY_PUBLIC, nil
	case bufpolicy.PolicyVisibilityPrivate:
		return policyv1beta1.PolicyVisibility_POLICY_VISIBILITY_PRIVATE, nil
	default:
		return 0, fmt.Errorf("unknown PolicyVisibility: %v", policyVisibility)
	}
}

func policyTypeToV1Beta1Proto(policyType bufpolicy.PolicyType) (policyv1beta1.PolicyType, error) {
	switch policyType {
	case bufpolicy.PolicyTypeCheck:
		return policyv1beta1.PolicyType_POLICY_TYPE_CHECK, nil
	default:
		return 0, fmt.Errorf("unknown PolicyType: %v", policyType)
	}
}

func v1beta1ProtoToDigestType(protoDigestType policyv1beta1.DigestType) (bufpolicy.DigestType, error) {
	digestType, ok := v1beta1ProtoDigestTypeToDigestType[protoDigestType]
	if !ok {
		return 0, fmt.Errorf("unknown policyv1beta1.DigestType: %v", protoDigestType)
	}
	return digestType, nil
}
