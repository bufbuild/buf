// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufmoduleapi

import (
	"fmt"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
)

var (
	digestTypeToProto = map[bufmodule.DigestType]modulev1beta1.DigestType{
		bufmodule.DigestTypeB4: modulev1beta1.DigestType_DIGEST_TYPE_B4,
		bufmodule.DigestTypeB5: modulev1beta1.DigestType_DIGEST_TYPE_B5,
	}
	protoToDigestType = map[modulev1beta1.DigestType]bufmodule.DigestType{
		modulev1beta1.DigestType_DIGEST_TYPE_B4: bufmodule.DigestTypeB4,
		modulev1beta1.DigestType_DIGEST_TYPE_B5: bufmodule.DigestTypeB5,
	}
)

// digestToProto converts the given Digest to a proto Digest.
func digestToProto(digest bufmodule.Digest) (*modulev1beta1.Digest, error) {
	protoDigestType, ok := digestTypeToProto[digest.Type()]
	// Technically we have already done this validation but just to be safe.
	if !ok {
		return nil, fmt.Errorf("unknown DigestType: %v", digest.Type())
	}
	protoDigest := &modulev1beta1.Digest{
		Type:  protoDigestType,
		Value: digest.Value(),
	}
	return protoDigest, nil
}

// protoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func protoToDigest(protoDigest *modulev1beta1.Digest) (bufmodule.Digest, error) {
	digestType, ok := protoToDigestType[protoDigest.Type]
	if !ok {
		return nil, fmt.Errorf("unknown proto Digest.Type: %v", protoDigest.Type)
	}
	bufcasDigest, err := bufcas.NewDigest(protoDigest.Value)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewDigest(digestType, bufcasDigest)
}
