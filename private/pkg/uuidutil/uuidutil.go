// Copyright 2020-2025 Buf Technologies, Inc.
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

package uuidutil

import (
	"fmt"

	"github.com/google/uuid"
)

// New returns a new random UUIDv4.
func New() (uuid.UUID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// ToDashless returns the uuid without dashes.
func ToDashless(id uuid.UUID) string {
	s := id.String()
	return s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:]
}

// FromString returns the uuid from the string.
//
// As opposed to uuid.FromString, this only accepts uuids with dashes.
// Always use this instead of uuid.FromString.
func FromString(s string) (uuid.UUID, error) {
	if len(s) != 36 {
		return uuid.Nil, fmt.Errorf("expected uuid to be of length 36 but was %d: %s", len(s), s)
	}
	return uuid.Parse(s)
}

// FromDashless returns the dashless uuid with dashes.
func FromDashless(dashless string) (uuid.UUID, error) {
	if len(dashless) != 32 {
		return uuid.Nil, fmt.Errorf("expected dashless uuid to be of length 32 but was %d: %s", len(dashless), dashless)
	}
	// FromString accepts both dashless and regular, we do this because we need to add our own FromString that does not accept dashless
	return FromString(dashless[0:8] + "-" + dashless[8:12] + "-" + dashless[12:16] + "-" + dashless[16:20] + "-" + dashless[20:])
}

// Validate determines if the given UUID string is valid.
func Validate(s string) error {
	_, err := FromString(s)
	return err
}

// ValidateDashless validates the dashless uuid is valid.
func ValidateDashless(dashless string) error {
	_, err := FromDashless(dashless)
	return err
}

// FromStringSlice returns a slice of uuids from the string slice.
//
// This only accepts uuids with dashes.
func FromStringSlice(s []string) ([]uuid.UUID, error) {
	var parsedUUIDS []uuid.UUID
	for _, stringID := range s {
		parsed, err := FromString(stringID)
		if err != nil {
			return nil, err
		}
		parsedUUIDS = append(parsedUUIDS, parsed)
	}
	return parsedUUIDS, nil
}
