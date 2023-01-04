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

package ulidutil

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// New creates a new ULID for the given timestamp.
func New(timestamp time.Time) (ulid.ULID, error) {
	return ulid.New(ulid.Timestamp(timestamp), rand.Reader)
}

// FromString returns the ULID from the string.
func FromString(s string) (ulid.ULID, error) {
	if len(s) != ulid.EncodedSize {
		return ulid.ULID{}, fmt.Errorf("expected ULID to be of length %d but was %d: %s", ulid.EncodedSize, len(s), s)
	}
	return ulid.ParseStrict(s)
}
