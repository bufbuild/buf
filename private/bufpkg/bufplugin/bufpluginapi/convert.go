// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufpluginapi

import (
	"fmt"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
)

var (
	v1beta1ProtoDigestTypeToDigestType = map[pluginv1beta1.DigestType]bufplugin.DigestType{
		pluginv1beta1.DigestType_DIGEST_TYPE_P1: bufplugin.DigestTypeP1,
	}
)

// V1Beta1ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func V1Beta1ProtoToDigest(protoDigest *pluginv1beta1.Digest) (bufplugin.Digest, error) {
	digestType, err := v1beta1ProtoToDigestType(protoDigest.Type)
	if err != nil {
		return nil, err
	}
	bufcasDigest, err := bufcas.NewDigest(protoDigest.Value)
	if err != nil {
		return nil, err
	}
	return bufplugin.NewDigest(digestType, bufcasDigest)
}

func v1beta1ProtoToDigestType(protoDigestType pluginv1beta1.DigestType) (bufplugin.DigestType, error) {
	digestType, ok := v1beta1ProtoDigestTypeToDigestType[protoDigestType]
	if !ok {
		return 0, fmt.Errorf("unknown pluginv1beta1.DigestType: %v", protoDigestType)
	}
	return digestType, nil
}
