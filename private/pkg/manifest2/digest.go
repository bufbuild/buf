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

package manifest2

import "encoding/hex"

type digest struct {
	digestType DigestType
	value      []byte
}

func newDigest(digestType DigestType, value []byte) *digest {
	return &digest{
		digestType: digestType,
		value:      value,
	}
}

func (d *digest) Type() DigestType {
	return d.digestType
}

func (d *digest) Value() []byte {
	return d.value
}

func (d *digest) String() string {
	return d.digestType.String() + ":" + hex.EncodeToString(d.value)
}
