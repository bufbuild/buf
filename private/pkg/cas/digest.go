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

package cas

import (
	"encoding/hex"
	"fmt"
)

type digest struct {
	digestType DigestType
	value      []byte
	// Cache as we call String pretty often.
	// We could do this lazily but not worth it.
	stringValue string
}

func newDigest(digestType DigestType, value []byte) (*digest, error) {
	switch digestType {
	case DigestTypeShake256:
		// No content, return a nil Digest.
		if len(value) == 0 {
			return nil, nil
		}
		if len(value) != shake256Length {
			return nil, fmt.Errorf("invalid %s Digest value: expected %d bytes, got %d", digestType.String(), shake256Length, len(value))
		}
		return &digest{
			digestType:  digestType,
			value:       value,
			stringValue: digestType.String() + ":" + hex.EncodeToString(value),
		}, nil
	default:
		return nil, fmt.Errorf("unknown DigestType: %v", digestType)
	}
}

func (d *digest) Type() DigestType {
	return d.digestType
}

func (d *digest) Value() []byte {
	return d.value
}

func (d *digest) String() string {
	return d.stringValue
}

func (*digest) isDigest() {}
