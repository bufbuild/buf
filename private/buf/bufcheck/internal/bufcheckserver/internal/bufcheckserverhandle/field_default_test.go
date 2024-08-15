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

package bufcheckserverhandle

import (
	"context"
	"math"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultsEqual(t *testing.T) {
	t.Parallel()
	// With an entry, all values are considered equal. But they are all
	// considered unequal to values in any other entry.
	values := map[string][]any{
		"zero": {
			int32(0), int64(0), uint32(0), uint64(0), float32(0), 0.0, false,
		},
		"one": {
			int32(1), int64(1), uint32(1), uint64(1), float32(1), 1.0, true,
		},
		"other-integer": {
			int32(456), int64(456), uint32(456), uint64(456), float32(456), 456.0,
		},
		"other-non-integer": {
			float32(-987.654), -987.654,
		},
		"nan": {
			float32(math.NaN()), math.NaN(), -float32(math.NaN()), -math.NaN(),
		},
		"positive-inf": {
			float32(math.Inf(1)), math.Inf(1),
		},
		"negative-inf": {
			float32(math.Inf(-1)), math.Inf(-1),
		},
		"other-string": {
			"foobar",
		},
	}
	for name, vals := range values {
		name, vals := name, vals
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, val := range vals {
				for other, otherVals := range values {
					for _, otherVal := range otherVals {
						default1 := fieldDefault{
							comparable: val, printable: val,
						}
						default2 := fieldDefault{
							comparable: otherVal, printable: otherVal,
						}
						if name == other {
							assert.True(t, defaultsEqual(default1, default2),
								"expected %v (%T) to be sufficiently equal default to %v (%T)",
								val, val, otherVal, otherVal)
						} else {
							assert.False(t, defaultsEqual(default1, default2),
								"expected %v (%T) to be NOT equal default to %v (%T)",
								val, val, otherVal, otherVal)
						}
					}
				}
			}
		})
	}
}

func TestGetDefault(t *testing.T) {
	t.Parallel()
	testFile := `
		syntax = "proto2";
		message A {
			optional int32 int32 = 1 [default=123];
			optional sint32 sint32 = 2 [default=123];
			optional uint32 uint32 = 3 [default=123];
			optional fixed32 fixed32 = 4 [default=123];
			optional sfixed32 sfixed32 = 5 [default=123];
			optional int64 int64 = 6 [default=123];
			optional sint64 sint64 = 7 [default=123];
			optional uint64 uint64 = 8 [default=123];
			optional fixed64 fixed64 = 9 [default=123];
			optional sfixed64 sfixed64 = 10 [default=123];
			optional float float = 11 [default=123.123];
			optional double double = 12 [default=123.123];
			optional bool bool = 13 [default=true];
			optional string string = 14 [default="xyz"];
			optional bytes bytes = 15 [default="xyz"];
			optional Enum enum = 16 [default=V123];
			optional A message = 17;
			repeated int32 repeated = 18;
			map<int32, int32> map = 19;
		}
		enum Enum {
			V0 = 0;
			V1 = 1;
			V123 = 123;
		}`
	compiler := &protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				"test.proto": testFile,
			}),
		},
	}
	results, err := compiler.Compile(context.Background(), "test.proto")
	require.NoError(t, err)
	msg := results[0].Messages().ByName("A")

	assert.Equal(t,
		fieldDefault{
			comparable: int32(123),
			printable:  int32(123),
		},
		getDefault(msg.Fields().ByName("int32")))
	assert.Equal(t,
		fieldDefault{
			comparable: int32(123),
			printable:  int32(123),
		},
		getDefault(msg.Fields().ByName("sint32")))
	assert.Equal(t,
		fieldDefault{
			comparable: uint32(123),
			printable:  uint32(123),
		},
		getDefault(msg.Fields().ByName("uint32")))
	assert.Equal(t,
		fieldDefault{
			comparable: uint32(123),
			printable:  uint32(123),
		},
		getDefault(msg.Fields().ByName("fixed32")))
	assert.Equal(t,
		fieldDefault{
			comparable: int32(123),
			printable:  int32(123),
		},
		getDefault(msg.Fields().ByName("sfixed32")))

	assert.Equal(t,
		fieldDefault{
			comparable: int64(123),
			printable:  int64(123),
		},
		getDefault(msg.Fields().ByName("int64")))
	assert.Equal(t,
		fieldDefault{
			comparable: int64(123),
			printable:  int64(123),
		},
		getDefault(msg.Fields().ByName("sint64")))
	assert.Equal(t,
		fieldDefault{
			comparable: uint64(123),
			printable:  uint64(123),
		},
		getDefault(msg.Fields().ByName("uint64")))
	assert.Equal(t,
		fieldDefault{
			comparable: uint64(123),
			printable:  uint64(123),
		},
		getDefault(msg.Fields().ByName("fixed64")))
	assert.Equal(t,
		fieldDefault{
			comparable: int64(123),
			printable:  int64(123),
		},
		getDefault(msg.Fields().ByName("sfixed64")))

	assert.Equal(t,
		fieldDefault{
			comparable: float32(123.123),
			printable:  float32(123.123),
		},
		getDefault(msg.Fields().ByName("float")))
	assert.Equal(t,
		fieldDefault{
			comparable: 123.123,
			printable:  123.123,
		},
		getDefault(msg.Fields().ByName("double")))
	assert.Equal(t,
		fieldDefault{
			comparable: true,
			printable:  true,
		},
		getDefault(msg.Fields().ByName("bool")))

	assert.Equal(t,
		fieldDefault{
			comparable: "xyz",
			printable:  `"xyz"`,
		},
		getDefault(msg.Fields().ByName("string")))
	assert.Equal(t,
		fieldDefault{
			comparable: "xyz",
			printable:  "[0x78,0x79,0x7A]",
		},
		getDefault(msg.Fields().ByName("bytes")))
	assert.Equal(t,
		fieldDefault{
			comparable: int32(123),
			printable:  "Enum.V123",
		},
		getDefault(msg.Fields().ByName("enum")))

	cannotHaveDefault := map[string]struct{}{
		"message":  {},
		"repeated": {},
		"map":      {},
	}
	for i := 0; i < msg.Fields().Len(); i++ {
		field := msg.Fields().Get(i)
		if _, nope := cannotHaveDefault[string(field.Name())]; nope {
			assert.False(t, canHaveDefault(field))
		} else {
			assert.True(t, canHaveDefault(field))
		}
	}
}
