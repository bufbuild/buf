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

package git

import (
	"encoding/hex"
	"fmt"
)

// hashLength is the length, in bytes, of digests/hashes in object format SHA1
const hashLength = 20

// hashHexLength is the length, in hexadecimal characters, of digests/hashes in object format SHA1
var hashHexLength = hex.EncodedLen(hashLength)

type hash struct {
	raw []byte
	hex string
}

func (i *hash) Raw() []byte {
	return i.raw
}

func (i *hash) Hex() string {
	return i.hex
}

func (i *hash) String() string {
	return i.hex
}

func newHashFromBytes(data []byte) (*hash, error) {
	if len(data) != hashLength {
		return nil, fmt.Errorf("hash is not %d bytes", hashLength)
	}
	dst := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(dst, data)
	return &hash{
		raw: data,
		hex: string(dst),
	}, nil
}

func parseHashFromHex(data string) (*hash, error) {
	if len(data) != hashHexLength {
		return nil, fmt.Errorf("hash is not %d characters", hashHexLength)
	}
	raw, err := hex.DecodeString(data)
	return &hash{
		raw: raw,
		hex: data,
	}, err
}
