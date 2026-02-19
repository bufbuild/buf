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

package buflsp_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestCELHover(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	testProtoPath, err := filepath.Abs("testdata/hover/cel_comprehensive.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	testCases := []struct {
		name     string
		line     uint32 // 0-indexed
		char     uint32 // Character offset where the token starts
		expected string // Full expected hover content
	}{
		// Keywords
		{
			name:     "keyword: this",
			line:     19,
			char:     17,
			expected: "**Special variable**\n\nRefers to the current message or field being validated.\n\nIn field-level rules, `this` refers to the field value.\nIn message-level rules, `this` refers to the entire message.",
		},
		{
			name:     "keyword: true",
			line:     24,
			char:     25,
			expected: "**Literal**: `true`\n\n**Type**: bool",
		},
		{
			name:     "keyword: false",
			line:     29,
			char:     25,
			expected: "**Literal**: `false`\n\n**Type**: bool",
		},
		{
			name:     "keyword: null",
			line:     34,
			char:     37, // Position of first "null" in full line: '    expression: "true ? this != '' : null != null"'
			expected: "**Literal**: `null`\n\n**Type**: value",
		},

		// Logical operators
		{
			name:     "operator: &&",
			line:     40,
			char:     22,
			expected: "**Operator**: `&&`\n\nlogically AND two boolean values. Errors and unknown values\nare valid inputs and will not halt evaluation.\n\n**Overloads**:\n- `bool && bool -> bool`",
		},
		{
			name:     "operator: ||",
			line:     45,
			char:     22,
			expected: "**Operator**: `||`\n\nlogically OR two boolean values. Errors and unknown values\nare valid inputs and will not halt evaluation.\n\n**Overloads**:\n- `bool || bool -> bool`",
		},
		{
			name:     "operator: !",
			line:     50,
			char:     17,
			expected: "**Operator**: `!`\n\nlogically negate a boolean value.\n\n**Overloads**:\n- `!bool -> bool`",
		},

		// Comparison operators
		{
			name:     "operator: ==",
			line:     56,
			char:     22,
			expected: "**Operator**: `==`\n\ncompare two values of the same type for equality\n\n**Overloads**:\n- `<A> == <A> -> bool`",
		},
		{
			name:     "operator: !=",
			line:     61,
			char:     22,
			expected: "**Operator**: `!=`\n\ncompare two values of the same type for inequality\n\n**Overloads**:\n- `<A> != <A> -> bool`",
		},
		{
			name:     "operator: <",
			line:     66,
			char:     22,
			expected: "**Operator**: `<`\n\ncompare two values and return true if the first value is\nless than the second\n\n**Overloads**:\n- `bool < bool -> bool`\n- `int < int -> bool`\n- `int < double -> bool`\n- `int < uint -> bool`\n- `uint < uint -> bool`\n- `uint < double -> bool`\n- `uint < int -> bool`\n- `double < double -> bool`\n- `double < int -> bool`\n- `double < uint -> bool`\n- `string < string -> bool`\n- `bytes < bytes -> bool`\n- `google.protobuf.Timestamp < google.protobuf.Timestamp -> bool`\n- `google.protobuf.Duration < google.protobuf.Duration -> bool`",
		},
		{
			name:     "operator: >",
			line:     71,
			char:     22,
			expected: "**Operator**: `>`\n\ncompare two values and return true if the first value is\ngreater than the second\n\n**Overloads**:\n- `bool > bool -> bool`\n- `int > int -> bool`\n- `int > double -> bool`\n- `int > uint -> bool`\n- `uint > uint -> bool`\n- `uint > double -> bool`\n- `uint > int -> bool`\n- `double > double -> bool`\n- `double > int -> bool`\n- `double > uint -> bool`\n- `string > string -> bool`\n- `bytes > bytes -> bool`\n- `google.protobuf.Timestamp > google.protobuf.Timestamp -> bool`\n- `google.protobuf.Duration > google.protobuf.Duration -> bool`",
		},
		{
			name:     "operator: <=",
			line:     76,
			char:     22,
			expected: "**Operator**: `<=`\n\ncompare two values and return true if the first value is\nless than or equal to the second\n\n**Overloads**:\n- `bool <= bool -> bool`\n- `int <= int -> bool`\n- `int <= double -> bool`\n- `int <= uint -> bool`\n- `uint <= uint -> bool`\n- `uint <= double -> bool`\n- `uint <= int -> bool`\n- `double <= double -> bool`\n- `double <= int -> bool`\n- `double <= uint -> bool`\n- `string <= string -> bool`\n- `bytes <= bytes -> bool`\n- `google.protobuf.Timestamp <= google.protobuf.Timestamp -> bool`\n- `google.protobuf.Duration <= google.protobuf.Duration -> bool`",
		},
		{
			name:     "operator: >=",
			line:     81,
			char:     22,
			expected: "**Operator**: `>=`\n\ncompare two values and return true if the first value is\ngreater than or equal to the second\n\n**Overloads**:\n- `bool >= bool -> bool`\n- `int >= int -> bool`\n- `int >= double -> bool`\n- `int >= uint -> bool`\n- `uint >= uint -> bool`\n- `uint >= double -> bool`\n- `uint >= int -> bool`\n- `double >= double -> bool`\n- `double >= int -> bool`\n- `double >= uint -> bool`\n- `string >= string -> bool`\n- `bytes >= bytes -> bool`\n- `google.protobuf.Timestamp >= google.protobuf.Timestamp -> bool`\n- `google.protobuf.Duration >= google.protobuf.Duration -> bool`",
		},

		// Arithmetic operators
		{
			name:     "operator: +",
			line:     87,
			char:     22,
			expected: "**Operator**: `+`\n\nadds two numeric values or concatenates two strings, bytes,\nor lists.\n\n**Overloads**:\n- `bytes + bytes -> bytes`\n- `double + double -> double`\n- `google.protobuf.Duration + google.protobuf.Duration -> google.protobuf.Duration`\n- `google.protobuf.Duration + google.protobuf.Timestamp -> google.protobuf.Timestamp`\n- `google.protobuf.Timestamp + google.protobuf.Duration -> google.protobuf.Timestamp`\n- `int + int -> int`\n- `list(<A>) + list(<A>) -> list(<A>)`\n- `string + string -> string`\n- `uint + uint -> uint`",
		},
		{
			name:     "operator: -",
			line:     92,
			char:     22,
			expected: "**Operator**: `-`\n\nsubtract two numbers, or two time-related values\n\n**Overloads**:\n- `double - double -> double`\n- `google.protobuf.Duration - google.protobuf.Duration -> google.protobuf.Duration`\n- `int - int -> int`\n- `google.protobuf.Timestamp - google.protobuf.Duration -> google.protobuf.Timestamp`\n- `google.protobuf.Timestamp - google.protobuf.Timestamp -> google.protobuf.Duration`\n- `uint - uint -> uint`",
		},
		{
			name:     "operator: *",
			line:     97,
			char:     22,
			expected: "**Operator**: `*`\n\nmultiply two numbers\n\n**Overloads**:\n- `double * double -> double`\n- `int * int -> int`\n- `uint * uint -> uint`",
		},
		{
			name:     "operator: /",
			line:     102,
			char:     22,
			expected: "**Operator**: `/`\n\ndivide two numbers\n\n**Overloads**:\n- `double / double -> double`\n- `int / int -> int`\n- `uint / uint -> uint`",
		},
		{
			name:     "operator: %",
			line:     107,
			char:     22,
			expected: "**Operator**: `%`\n\ncompute the modulus of one integer into another\n\n**Overloads**:\n- `int % int -> int`\n- `uint % uint -> uint`",
		},

		// String functions (upstream docs)
		{
			name:     "function: contains",
			line:     113,
			char:     23,
			expected: "`contains`\n\ntest whether a string contains a substring\n\n**Overloads**:\n- `string.contains(string) -> bool`\n- `bytes.contains(bytes) -> bool`",
		},
		{
			name:     "function: startsWith",
			line:     118,
			char:     23,
			expected: "`startsWith`\n\ntest whether a string starts with a substring prefix\n\n**Overloads**:\n- `string.startsWith(string) -> bool`\n- `bytes.startsWith(bytes) -> bool`",
		},
		{
			name:     "function: endsWith",
			line:     123,
			char:     23,
			expected: "`endsWith`\n\ntest whether a string ends with a substring suffix\n\n**Overloads**:\n- `string.endsWith(string) -> bool`\n- `bytes.endsWith(bytes) -> bool`",
		},
		{
			name:     "function: matches",
			line:     128,
			char:     23,
			expected: "`matches`\n\ntest whether a string matches an RE2 regular expression\n\n**Overloads**:\n- `matches(string, string) -> bool`\n- `string.matches(string) -> bool`",
		},

		// Collection functions (upstream docs)
		{
			name:     "function: size",
			line:     134,
			char:     23,
			expected: "`size`\n\ncompute the size of a list or map, the number of characters in a string,\nor the number of bytes in a sequence\n\n**Overloads**:\n- `size(bytes) -> int`\n- `bytes.size() -> int`\n- `size(list(<A>)) -> int`\n- `list(<A>).size() -> int`\n- `size(map(<A>, <B>)) -> int`\n- `map(<A>, <B>).size() -> int`\n- `size(string) -> int`\n- `string.size() -> int`",
		},
		{
			name:     "operator: in",
			line:     139,
			char:     26,
			expected: "**Operator**: `in`\n\ntest whether a value exists in a list, or a key exists in a map\n\n**Overloads**:\n- `<A> in list(<A>) -> bool`\n- `<A> in map(<A>, <B>) -> bool`",
		},

		// Type conversion functions (upstream docs)
		{
			name:     "function: int",
			line:     145,
			char:     17,
			expected: "`int`\n\nconvert a value to an int\n\n**Overloads**:\n- `int(int) -> int`\n- `int(double) -> int`\n- `int(google.protobuf.Duration) -> int`\n- `int(string) -> int`\n- `int(google.protobuf.Timestamp) -> int`\n- `int(uint) -> int`",
		},
		{
			name:     "function: uint",
			line:     150,
			char:     17,
			expected: "`uint`\n\nconvert a value to a uint\n\n**Overloads**:\n- `uint(uint) -> uint`\n- `uint(double) -> uint`\n- `uint(int) -> uint`\n- `uint(string) -> uint`",
		},
		{
			name:     "function: double",
			line:     155,
			char:     17,
			expected: "`double`\n\nconvert a value to a double\n\n**Overloads**:\n- `double(double) -> double`\n- `double(int) -> double`\n- `double(string) -> double`\n- `double(uint) -> double`",
		},
		{
			name:     "function: string",
			line:     160,
			char:     17,
			expected: "`string`\n\nconvert a value to a string\n\n**Overloads**:\n- `string(string) -> string`\n- `string(bool) -> string`\n- `string(bytes) -> string`\n- `string(double) -> string`\n- `string(google.protobuf.Duration) -> string`\n- `string(int) -> string`\n- `string(google.protobuf.Timestamp) -> string`\n- `string(uint) -> string`",
		},
		{
			name:     "function: bytes",
			line:     165,
			char:     17,
			expected: "`bytes`\n\nconvert a value to bytes\n\n**Overloads**:\n- `bytes(bytes) -> bytes`\n- `bytes(string) -> bytes`",
		},
		{
			name:     "function: timestamp",
			line:     170,
			char:     17,
			expected: "`timestamp`\n\nconvert a value to a google.protobuf.Timestamp\n\n**Overloads**:\n- `timestamp(google.protobuf.Timestamp) -> google.protobuf.Timestamp`\n- `timestamp(int) -> google.protobuf.Timestamp`\n- `timestamp(string) -> google.protobuf.Timestamp`",
		},
		{
			name:     "function: duration",
			line:     175,
			char:     17,
			expected: "`duration`\n\nconvert a value to a google.protobuf.Duration\n\n**Overloads**:\n- `duration(google.protobuf.Duration) -> google.protobuf.Duration`\n- `duration(int) -> google.protobuf.Duration`\n- `duration(string) -> google.protobuf.Duration`",
		},
		{
			name:     "function: type",
			line:     180,
			char:     17,
			expected: "`type`\n\nconvert a value to its type identifier\n\n**Overloads**:\n- `type(<A>) -> type`",
		},
		{
			name:     "function: dyn",
			line:     185,
			char:     17,
			expected: "`dyn`\n\nindicate that the type is dynamic for type-checking purposes\n\n**Overloads**:\n- `dyn(<A>) -> dyn`",
		},

		// Timestamp functions (upstream docs)
		{
			name:     "function: getFullYear",
			line:     191,
			char:     23,
			expected: "`getFullYear`\n\nget the 0-based full year from a timestamp, UTC unless an IANA timezone is specified.\n\n**Overloads**:\n- `google.protobuf.Timestamp.getFullYear() -> int`\n- `google.protobuf.Timestamp.getFullYear(string) -> int`",
		},
		{
			name:     "function: getMonth",
			line:     196,
			char:     23,
			expected: "`getMonth`\n\nget the 0-based month from a timestamp, UTC unless an IANA timezone is specified.\n\n**Overloads**:\n- `google.protobuf.Timestamp.getMonth() -> int`\n- `google.protobuf.Timestamp.getMonth(string) -> int`",
		},
		{
			name:     "function: getDate",
			line:     201,
			char:     23,
			expected: "`getDate`\n\nget the 1-based day of the month from a timestamp, UTC unless an IANA timezone is specified.\n\n**Overloads**:\n- `google.protobuf.Timestamp.getDate() -> int`\n- `google.protobuf.Timestamp.getDate(string) -> int`",
		},
		{
			name:     "function: getHours",
			line:     206,
			char:     23,
			expected: "`getHours`\n\nget the hours portion from a timestamp, or convert a duration to hours\n\n**Overloads**:\n- `google.protobuf.Timestamp.getHours() -> int`\n- `google.protobuf.Timestamp.getHours(string) -> int`\n- `google.protobuf.Duration.getHours() -> int`",
		},
		{
			name:     "function: getMinutes",
			line:     211,
			char:     23,
			expected: "`getMinutes`\n\nget the minutes portion from a timestamp, or convert a duration to minutes\n\n**Overloads**:\n- `google.protobuf.Timestamp.getMinutes() -> int`\n- `google.protobuf.Timestamp.getMinutes(string) -> int`\n- `google.protobuf.Duration.getMinutes() -> int`",
		},
		{
			name:     "function: getSeconds",
			line:     216,
			char:     23,
			expected: "`getSeconds`\n\nget the seconds portion from a timestamp, or convert a duration to seconds\n\n**Overloads**:\n- `google.protobuf.Timestamp.getSeconds() -> int`\n- `google.protobuf.Timestamp.getSeconds(string) -> int`\n- `google.protobuf.Duration.getSeconds() -> int`",
		},
		{
			name:     "function: getDayOfWeek",
			line:     221,
			char:     23,
			expected: "`getDayOfWeek`\n\nget the 0-based day of the week from a timestamp, UTC unless an IANA timezone is specified.\n\n**Overloads**:\n- `google.protobuf.Timestamp.getDayOfWeek() -> int`\n- `google.protobuf.Timestamp.getDayOfWeek(string) -> int`",
		},

		// Duration functions (upstream docs)
		{
			name:     "function: duration.getSeconds",
			line:     227,
			char:     23,
			expected: "`getSeconds`\n\nget the seconds portion from a timestamp, or convert a duration to seconds\n\n**Overloads**:\n- `google.protobuf.Timestamp.getSeconds() -> int`\n- `google.protobuf.Timestamp.getSeconds(string) -> int`\n- `google.protobuf.Duration.getSeconds() -> int`",
		},

		// Macros (upstream docs with examples)
		{
			name:     "macro: has",
			line:     233,
			char:     17,
			expected: "**Macro**: `has`\n\ncheck a protocol buffer message for the presence of a field, or check a map\nfor the presence of a string key.\nOnly map accesses using the select notation are supported.\n\n**Examples**:\n```cel\n// true if the 'address' field exists in the 'user' message\nhas(user.address)\n```\n```cel\n// test whether the 'key_name' is set on the map which defines it\nhas({'key_name': 'value'}.key_name) // true\n```\n```cel\n// test whether the 'id' field is set to a non-default value on the Expr{} message literal\nhas(Expr{}.id) // false\n```",
		},
		{
			name:     "macro: all",
			line:     238,
			char:     22,
			expected: "**Macro**: `all`\n\ntests whether all elements in the input list or all keys in a map\nsatisfy the given predicate. The all macro behaves in a manner consistent with\nthe Logical AND operator including in how it absorbs errors and short-circuits.\n\n**Examples**:\n```cel\n[1, 2, 3].all(x, x > 0) // true\n```\n```cel\n[1, 2, 0].all(x, x > 0) // false\n```\n```cel\n['apple', 'banana', 'cherry'].all(fruit, fruit.size() > 3) // true\n```\n```cel\n[3.14, 2.71, 1.61].all(num, num < 3.0) // false\n```\n```cel\n{'a': 1, 'b': 2, 'c': 3}.all(key, key != 'b') // false\n```\n```cel\n// an empty list or map as the range will result in a trivially true result\n[].all(x, x > 0) // true\n```",
		},
		{
			name:     "macro: exists",
			line:     243,
			char:     22,
			expected: "**Macro**: `exists`\n\ntests whether any value in the list or any key in the map\nsatisfies the predicate expression. The exists macro behaves in a manner\nconsistent with the Logical OR operator including in how it absorbs errors and\nshort-circuits.\n\n**Examples**:\n```cel\n[1, 2, 3].exists(i, i % 2 != 0) // true\n```\n```cel\n[0, -1, 5].exists(num, num < 0) // true\n```\n```cel\n{'x': 'foo', 'y': 'bar'}.exists(key, key.startsWith('z')) // false\n```\n```cel\n// an empty list or map as the range will result in a trivially false result\n[].exists(i, i > 0) // false\n```\n```cel\n// test whether a key name equalling 'iss' exists in the map and the\n// value contains the substring 'cel.dev'\n// tokens = {'sub': 'me', 'iss': 'https://issuer.cel.dev'}\ntokens.exists(k, k == 'iss' && tokens[k].contains('cel.dev'))\n```",
		},
		{
			name:     "macro: exists_one",
			line:     248,
			char:     22,
			expected: "**Macro**: `exists_one`\n\ntests whether exactly one list element or map key satisfies\nthe predicate expression. This macro does not short-circuit in order to remain\nconsistent with logical operators being the only operators which can absorb\nerrors within CEL.\n\n**Examples**:\n```cel\n[1, 2, 2].exists_one(i, i < 2) // true\n```\n```cel\n{'a': 'hello', 'aa': 'hellohello'}.exists_one(k, k.startsWith('a')) // false\n```\n```cel\n[1, 2, 3, 4].exists_one(num, num % 2 == 0) // false\n```\n```cel\n// ensure exactly one key in the map ends in @acme.co\n{'wiley@acme.co': 'coyote', 'aa@milne.co': 'bear'}.exists_one(k, k.endsWith('@acme.co')) // true\n```",
		},
		{
			name:     "macro: map",
			line:     253,
			char:     22,
			expected: "**Macro**: `map`\n\nthe three-argument form of map transforms all elements in the input range.\n\n**Examples**:\n```cel\n[1, 2, 3].map(x, x * 2) // [2, 4, 6]\n```\n```cel\n[5, 10, 15].map(x, x / 5) // [1, 2, 3]\n```\n```cel\n['apple', 'banana'].map(fruit, fruit.upperAscii()) // ['APPLE', 'BANANA']\n```\n```cel\n// Combine all map key-value pairs into a list\n{'hi': 'you', 'howzit': 'bruv'}.map(k,\n    k + \":\" + {'hi': 'you', 'howzit': 'bruv'}[k]) // ['hi:you', 'howzit:bruv']\n```",
		},
		{
			name:     "macro: filter",
			line:     258,
			char:     22,
			expected: "**Macro**: `filter`\n\nreturns a list containing only the elements from the input list\nthat satisfy the given predicate\n\n**Examples**:\n```cel\n[1, 2, 3].filter(x, x > 1) // [2, 3]\n```\n```cel\n['cat', 'dog', 'bird', 'fish'].filter(pet, pet.size() == 3) // ['cat', 'dog']\n```\n```cel\n[{'a': 10, 'b': 5, 'c': 20}].map(m, m.filter(key, m[key] > 10)) // [['c']]\n```\n```cel\n// filter a list to select only emails with the @cel.dev suffix\n['alice@buf.io', 'tristan@cel.dev'].filter(v, v.endsWith('@cel.dev')) // ['tristan@cel.dev']\n```\n```cel\n// filter a map into a list, selecting only the values for keys that start with 'http-auth'\n{'http-auth-agent': 'secret', 'user-agent': 'mozilla'}.filter(k,\n     k.startsWith('http-auth')) // ['secret']\n```",
		},

		// Protovalidate extension functions (using upstream docs)
		{
			name:     "function: isEmail",
			line:     264,
			char:     22, // Position of "isEmail" in "this.isEmail()"
			expected: "`isEmail`\n\n**Overloads**:\n- `string.isEmail() -> bool`",
		},
		{
			name:     "function: isHostname",
			line:     269,
			char:     22, // Position of "isHostname" in "this.isHostname()"
			expected: "`isHostname`\n\n**Overloads**:\n- `string.isHostname() -> bool`",
		},
		{
			name:     "function: isIp",
			line:     274,
			char:     22, // Position of "isIp" in "this.isIp()"
			expected: "`isIp`\n\n**Overloads**:\n- `string.isIp() -> bool`\n- `string.isIp(int) -> bool`",
		},
		{
			name:     "function: isIpPrefix",
			line:     279,
			char:     22, // Position of "isIpPrefix" in "this.isIpPrefix()"
			expected: "`isIpPrefix`\n\n**Overloads**:\n- `string.isIpPrefix() -> bool`\n- `string.isIpPrefix(int) -> bool`\n- `string.isIpPrefix(bool) -> bool`\n- `string.isIpPrefix(int, bool) -> bool`",
		},
		{
			name:     "function: isUri",
			line:     284,
			char:     22, // Position of "isUri" in "this.isUri()"
			expected: "`isUri`\n\n**Overloads**:\n- `string.isUri() -> bool`",
		},
		{
			name:     "function: isUriRef",
			line:     289,
			char:     22, // Position of "isUriRef" in "this.isUriRef()"
			expected: "`isUriRef`\n\n**Overloads**:\n- `string.isUriRef() -> bool`",
		},
		{
			name:     "function: isHostAndPort",
			line:     294,
			char:     22, // Position of "isHostAndPort" in "this.isHostAndPort(true)"
			expected: "`isHostAndPort`\n\n**Overloads**:\n- `string.isHostAndPort(bool) -> bool`",
		},

		// Field access (proto field resolution)
		{
			name:     "field: city",
			line:     300,
			char:     23,
			expected: "**Field**: `city`\n\n**Proto Field**: `test.cel.v1.Address.city`\n\n**Field Number**: 1\n\n**Proto Type**: `string`\n\nencoded (hex): `0A` (1 byte)",
		},

		// Literals
		{
			name:     "literal: string",
			line:     306,
			char:     26,
			expected: "**Literal**: `'literal'`\n\n**Type**: string",
		},
		{
			name:     "literal: int",
			line:     311,
			char:     25,
			expected: "**Literal**: `42`\n\n**Type**: int64",
		},
		{
			name:     "literal: double",
			line:     316,
			char:     25,
			expected: "**Literal**: `3.14`\n\n**Type**: double",
		},
		{
			name:     "literal: bool",
			line:     321,
			char:     25,
			expected: "**Literal**: `true`\n\n**Type**: bool",
		},

		// Comprehension variables
		{
			name:     "variable: item",
			line:     327,
			char:     32,
			expected: "**Variable**: `item`\n\nLoop variable from comprehension.",
		},

		// Ternary operator
		{
			name:     "operator: ?",
			line:     34,
			char:     22,
			expected: "**Operator**: `?`\n\nThe ternary operator tests a boolean predicate and returns the left-hand side (truthy) expression if true, or the right-hand side (falsy) expression if false\n\n**Overloads**:\n- `bool ? <T> : <T> -> <T>`",
		},

		// Unicode: emoji are 2 UTF-16 code units but 1 rune each.
		// These tests verify correct offset handling past non-ASCII content.
		// '🎉😀' == this  (expression on line 335, 0-indexed: 334)
		//  ^  ^       ^--- this at UTF-16 char 27
		//  |  😀 at UTF-16 chars 20-21
		//  🎉 at UTF-16 chars 18-19
		//  '🎉😀' is at UTF-16 chars 17-22; == at 24; space at 26; this at 27
		{
			name:     "unicode: string literal (first emoji)",
			line:     334,
			char:     18,
			expected: "**Literal**: `'🎉😀'`\n\n**Type**: string",
		},
		{
			name:     "unicode: string literal (second emoji)",
			line:     334,
			char:     20,
			expected: "**Literal**: `'🎉😀'`\n\n**Type**: string",
		},
		{
			name:     "unicode: == operator after emoji",
			line:     334,
			char:     24,
			expected: "**Operator**: `==`\n\ncompare two values of the same type for equality\n\n**Overloads**:\n- `<A> == <A> -> bool`",
		},
		{
			name:     "unicode: this after emoji",
			line:     334,
			char:     27,
			expected: "**Special variable**\n\nRefers to the current message or field being validated.\n\nIn field-level rules, `this` refers to the field value.\nIn message-level rules, `this` refers to the entire message.",
		},

		// Unicode: CJK characters are 3 UTF-8 bytes but 1 UTF-16 code unit and 1 rune —
		// a third point in the UTF-8/UTF-16/rune space (vs. emoji: 4/2/1, ASCII: 1/1/1).
		// '中文' == this  (expression on line 343, 0-indexed: 342)
		//  ^^        ^--- this at UTF-16 char 25
		//  |文 at UTF-16 char 19 (1 code unit, 3 UTF-8 bytes)
		//  中 at UTF-16 char 18 (1 code unit, 3 UTF-8 bytes)
		//  '中文' is at UTF-16 chars 17-20; == at 22; this at 25
		{
			name:     "unicode: CJK string literal",
			line:     342,
			char:     18,
			expected: "**Literal**: `'中文'`\n\n**Type**: string",
		},
		{
			name:     "unicode: == operator after CJK",
			line:     342,
			char:     22,
			expected: "**Operator**: `==`\n\ncompare two values of the same type for equality\n\n**Overloads**:\n- `<A> == <A> -> bool`",
		},
		{
			name:     "unicode: this after CJK",
			line:     342,
			char:     25,
			expected: "**Special variable**\n\nRefers to the current message or field being validated.\n\nIn field-level rules, `this` refers to the field value.\nIn message-level rules, `this` refers to the entire message.",
		},

		// Unicode: mixed emoji (4 UTF-8 bytes, 2 UTF-16 code units) and CJK (3 UTF-8
		// bytes, 1 UTF-16 code unit) in a single expression, verifying that the per-rune
		// offset walk handles interleaved character widths correctly.
		// '🎉中' == this  (expression on line 350, 0-indexed: 349)
		//  ^ ^        ^--- this at UTF-16 char 26
		//  | 中 at UTF-16 char 20 (1 code unit, 3 UTF-8 bytes)
		//  🎉 at UTF-16 chars 18-19 (2 code units, 4 UTF-8 bytes)
		//  '🎉中' is at UTF-16 chars 17-21; == at 23; this at 26
		{
			name:     "unicode: mixed literal (emoji part)",
			line:     349,
			char:     18,
			expected: "**Literal**: `'🎉中'`\n\n**Type**: string",
		},
		{
			name:     "unicode: mixed literal (CJK part)",
			line:     349,
			char:     20,
			expected: "**Literal**: `'🎉中'`\n\n**Type**: string",
		},
		{
			name:     "unicode: == operator after mixed",
			line:     349,
			char:     23,
			expected: "**Operator**: `==`\n\ncompare two values of the same type for equality\n\n**Overloads**:\n- `<A> == <A> -> bool`",
		},
		{
			name:     "unicode: this after mixed",
			line:     349,
			char:     26,
			expected: "**Special variable**\n\nRefers to the current message or field being validated.\n\nIn field-level rules, `this` refers to the field value.\nIn message-level rules, `this` refers to the entire message.",
		},

		// Multi-line: CEL expression split across two adjacent proto string literals,
		// which protocompile concatenates into one value. Hover targets in the second
		// segment exercise the multi-literal walk in createCELSpan.
		// Line 357 (0-indexed): `    expression: "this.size() > 0"`
		// Line 358 (0-indexed): `                " && this != ''"`
		//                          ^   ^--- this at UTF-16 char 21
		//                          && at UTF-16 char 18
		{
			name:     "multiline: && in second segment",
			line:     357,
			char:     18,
			expected: "**Operator**: `&&`\n\nlogically AND two boolean values. Errors and unknown values\nare valid inputs and will not halt evaluation.\n\n**Overloads**:\n- `bool && bool -> bool`",
		},
		{
			name:     "multiline: this in second segment",
			line:     357,
			char:     21,
			expected: "**Special variable**\n\nRefers to the current message or field being validated.\n\nIn field-level rules, `this` refers to the field value.\nIn message-level rules, `this` refers to the entire message.",
		},

		// Escape-sequence test: proto \n (2 source bytes, 1 CEL byte) placed
		// immediately before a single-byte token. Without escape-aware offset
		// handling the computed celOffset lands on whitespace and hover returns
		// nil; with the fix it lands on '?'.
		// Line 366 (0-indexed): `    expression: "true\n? this : false"`
		//                                                  ^--- '?' at column 23
		{
			name:     "escape: ? after proto \\n",
			line:     366,
			char:     23,
			expected: "**Operator**: `?`\n\nThe ternary operator tests a boolean predicate and returns the left-hand side (truthy) expression if true, or the right-hand side (falsy) expression if false\n\n**Overloads**:\n- `bool ? <T> : <T> -> <T>`",
		},

		// Message-level CEL rules
		{
			name:     "message-level: this",
			line:     377,
			char:     17,
			expected: "**Special variable**\n\nRefers to the current message or field being validated.\n\nIn field-level rules, `this` refers to the field value.\nIn message-level rules, `this` refers to the entire message.",
		},
		{
			name:     "message-level: &&",
			line:     377,
			char:     33,
			expected: "**Operator**: `&&`\n\nlogically AND two boolean values. Errors and unknown values\nare valid inputs and will not halt evaluation.\n\n**Overloads**:\n- `bool && bool -> bool`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var hoverResult *protocol.Hover
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.char},
				},
			}, &hoverResult)
			require.NoError(t, err)
			require.NotNil(t, hoverResult, "Expected hover result at Line:%d Char:%d", tc.line, tc.char)
			require.NotEmpty(t, hoverResult.Contents.Value, "Expected non-empty hover content at Line:%d Char:%d", tc.line, tc.char)

			assert.Equal(t, tc.expected, hoverResult.Contents.Value,
				"Hover content mismatch at Line:%d Char:%d", tc.line, tc.char)
		})
	}
}
