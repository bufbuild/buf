// Copyright 2020-2022 Buf Technologies, Inc.
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

package manifest

import (
	"fmt"

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"golang.org/x/crypto/sha3"
)

var hashKindToDigestType = map[modulev1alpha1.HashKind]DigestType{
	modulev1alpha1.HashKind_HASH_KIND_SHAKE256: DigestTypeShake256,
}

// NewDigestFromBlobHash maps a module Hash to a digest.
func NewDigestFromBlobHash(hash *modulev1alpha1.Hash) (*Digest, error) {
	if hash == nil {
		return nil, fmt.Errorf("nil hash")
	}
	dType, ok := hashKindToDigestType[hash.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported hash kind: %s", hash.Kind.String())
	}
	return NewDigestFromBytes(dType, hash.Digest)
}

// NewBlobFromBytes creates a module Blob from in-memory content.
func NewBlobFromBytes(content []byte) *modulev1alpha1.Blob {
	blob := &modulev1alpha1.Blob{
		Hash: &modulev1alpha1.Hash{
			Kind:   modulev1alpha1.HashKind_HASH_KIND_SHAKE256,
			Digest: make([]byte, 64),
		},
		Content: content,
	}
	sha3.ShakeSum256(blob.Hash.Digest, content)
	return blob
}
