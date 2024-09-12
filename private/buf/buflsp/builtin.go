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

// Data for the built-in types.

package buflsp

// builtinDocs contains documentation for the built-in types, to display in hover inlays.
var builtinDocs = map[string][]string{
	"int32": {
		"A 32-bit integer (varint encoding).",
		"",
		"Values of this type range between `-2147483648` and `2147483647`.",
		"Beware that negative values are encoded as five bytes on the wire!",
	},
	"int64": {
		"A 64-bit integer (varint encoding).",
		"",
		"Values of this type range between `-9223372036854775808` and `9223372036854775807`.",
		"Beware that negative values are encoded as ten bytes on the wire!",
	},

	"uint32": {
		"A 32-bit unsigned integer (varint encoding).",
		"",
		"Values of this type range between `0` and `4294967295`.",
	},
	"uint64": {
		"A 64-bit unsigned integer (varint encoding).",
		"",
		"Values of this type range between `0` and `18446744073709551615`.",
	},

	"sint32": {
		"A 32-bit integer (ZigZag encoding).",
		"",
		"Values of this type range between `-2147483648` and `2147483647`.",
	},
	"sint64": {
		"A 64-bit integer (ZigZag encoding).",
		"",
		"Values of this type range between `-9223372036854775808` and `9223372036854775807`.",
	},

	"fixed32": {
		"A 32-bit unsigned integer (4-byte encoding).",
		"",
		"Values of this type range between `0` and `4294967295`.",
	},
	"fixed64": {
		"A 64-bit unsigned integer (8-byte encoding).",
		"",
		"Values of this type range between `0` and `18446744073709551615`.",
	},

	"sfixed32": {
		"A 32-bit integer (4-byte encoding).",
		"",
		"Values of this type range between `-2147483648` and `2147483647`.",
	},
	"sfixed64": {
		"A 64-bit integer (8-byte encoding).",
		"",
		"Values of this type range between `-9223372036854775808` and `9223372036854775807`.",
	},

	"float": {
		"A single-precision floating point number (IEEE-745.2008 binary32).",
	},
	"double": {
		"A double-precision floating point number (IEEE-745.2008 binary64).",
	},

	"string": {
		"A string of text.",
		"",
		"Stores at most 4GB of text. Intended to be UTF-8 encoded Unicode; use `bytes` if you need other encodings.",
	},
	"bytes": {
		"A blob of arbitrary bytes.",
		"",
		"Stores at most 4GB of binary data. Encoded as base64 in JSON.",
	},

	"bool": {
		"A Boolean value: `true` or `false`.",
		"",
		"Encoded as a single byte: `0x00` or `0xff` (all non-zero bytes decode to `true`).",
	},

	"default": {
		"A magic option that specifies the field's default value.",
		"",
		"Unlike every other option on a field, this does not have a corresponding field in",
		"`google.protobuf.FieldOptions`; it is implemented by compiler magic.",
	},
}
