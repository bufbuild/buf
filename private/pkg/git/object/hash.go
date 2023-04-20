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

package object

import (
	"encoding/hex"
)

// ID represents an Object Identifier. Object Identifiers are hashes of the
// object's content. There are two hashes in service: a SHAttered-resistant
// SHA-1 and SHA-256. Either hash type can be treated as an opaque identifier.
type ID []byte

// String produces a hex encoded form of ID.
func (d ID) String() string {
	return string(encode(d))
}

// MarshalText produces a hex encoded form of ID.
func (d ID) MarshalText() ([]byte, error) {
	return encode(d), nil
}

// UnmarshalText consumed a hex encoded form of ID.
func (d *ID) UnmarshalText(txt []byte) error {
	decode, err := hex.DecodeString(string(txt))
	*d = decode
	return err
}

// MarshalBinary produces a binary form of ID.
func (d ID) MarshalBinary() ([]byte, error) {
	return []byte(d), nil
}

// UnmarshalBinary consumed a binary form of ID.
func (d *ID) UnmarshalBinary(data []byte) error {
	*d = data
	return nil
}

func encode(d ID) []byte {
	dst := make([]byte, hex.EncodedLen(len(d)))
	hex.Encode(dst, d)
	return dst
}
