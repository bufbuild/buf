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

package gitobject

import (
	"encoding/hex"
	"fmt"
)

// idLength is the length, in bytes, of digests/IDs in object format SHA1
const idLength = 20

// idLength is the length, in hexadecimal characters, of digests/IDs in object format SHA1
var idHexLength = hex.EncodedLen(idLength)

type id struct {
	raw []byte
	hex string
}

func (i *id) Raw() []byte {
	return i.raw
}

func (i *id) Hex() string {
	return i.hex
}

func (i *id) String() string {
	return i.hex
}

func newObjectIDFromBytes(data []byte) (*id, error) {
	if len(data) != idLength {
		return nil, fmt.Errorf("ID is not %d bytes", idLength)
	}
	dst := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(dst, data)
	return &id{
		raw: data,
		hex: string(dst),
	}, nil
}

func parseObjectIDFromHex(data string) (*id, error) {
	if len(data) != idHexLength {
		return nil, fmt.Errorf("ID is not %d characters", idHexLength)
	}
	raw, err := hex.DecodeString(data)
	return &id{
		raw: raw,
		hex: data,
	}, err
}
