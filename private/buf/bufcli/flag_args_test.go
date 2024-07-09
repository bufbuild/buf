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

package bufcli

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestStringPointerFlagSet(t *testing.T) {
	t.Parallel()
	expected := "foo"
	testParseStringPointer(t, "test-flag-name", &expected, "--test-flag-name", "foo")
}

func TestStringPointerFlagSetEmpty(t *testing.T) {
	t.Parallel()
	expected := ""
	testParseStringPointer(t, "test-flag-name", &expected, "--test-flag-name", "")
}

func TestStringPointerFlagNotSet(t *testing.T) {
	t.Parallel()
	testParseStringPointer(t, "test-flag-name", nil)
}

func testParseStringPointer(t *testing.T, flagName string, expectedResult *string, args ...string) {
	var stringPointer *string
	flagSet := pflag.NewFlagSet("test flag set", pflag.ContinueOnError)
	BindStringPointer(flagSet, flagName, &stringPointer, "test usage")
	err := flagSet.Parse(args)
	require.NoError(t, err)
	require.Equal(t, expectedResult, stringPointer)
}
