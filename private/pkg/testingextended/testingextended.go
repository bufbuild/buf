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

package testingextended

import (
	"flag"
	"testing"
	"time"
)

// SkipIfShort skips the test if testing.short is set.
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

// GetTestTimeout returns the value of the go test -timeout flag.
func GetTestTimeout(t *testing.T) time.Duration {
	if !flag.Parsed() {
		t.Fatal("unable to read testing timeout flag as flags have not been parsed")
	}

	// default is 10m to match the default timeout for go test
	timeout := 10 * time.Minute

	// If the test.timeout flag is set, use it.
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "test.timeout" {
			// It's fine if this panics. We expect to be in a test, and this should
			// be covered by the Go 1 compatibility promise.
			timeout = f.Value.(flag.Getter).Get().(time.Duration)
		}
	})
	return timeout
}
