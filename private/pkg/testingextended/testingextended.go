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

package testingextended

import (
	"testing"
	"time"
)

// SkipIfShort skips the test if testing.short is set.
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

// GetTestTimeout returns the time remaining until the test times out or 10m if the test is not set to timeout.
func GetTestTimeout(t *testing.T) time.Duration {
	if deadline, ok := t.Deadline(); ok && !deadline.IsZero() {
		return time.Until(deadline)
	}
	return 10 * time.Minute
}
