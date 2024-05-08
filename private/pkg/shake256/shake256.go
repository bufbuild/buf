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

// Package shake256 provides simple utilities around shake256 digests.
package shake256

import (
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"golang.org/x/crypto/sha3"
)

const shake256Length = 64

// Digest is a shake256 digest.
type Digest interface {
	Value() []byte

	isDigest()
}

// NewDigest returns a new Digest for the value.
func NewDigest(value []byte) (Digest, error) {
	return newDigest(value)
}

// NewDigest returns a new Digest for the content read from the Reader.
func NewDigestForContent(reader io.Reader) (Digest, error) {
	shakeHash := sha3.NewShake256()
	// TODO FUTURE: remove in the future, this should have no effect
	shakeHash.Reset()
	if _, err := io.Copy(shakeHash, reader); err != nil {
		return nil, err
	}
	value := make([]byte, shake256Length)
	if _, err := shakeHash.Read(value); err != nil {
		// sha3.ShakeHash never errors or short reads. Something horribly wrong
		// happened if your computer ended up here.
		return nil, err
	}
	return newDigest(value)
}

// *** PRIVATE ***

type digest struct {
	value []byte
}

func newDigest(value []byte) (*digest, error) {
	if len(value) != shake256Length {
		return nil, fmt.Errorf("invalid shake256 digest value: expected %d bytes, got %d", shake256Length, len(value))
	}
	return &digest{
		value: value,
	}, nil
}

func (d *digest) Value() []byte {
	return slicesext.Copy(d.value)
}

func (*digest) isDigest() {}
