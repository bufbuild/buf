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

package bufbreakingcheck

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type fieldDefault struct {
	comparable any
	printable  any
}

func (f fieldDefault) isZero() bool {
	return reflect.ValueOf(f.comparable).IsZero()
}

func getDefault(descriptor protoreflect.FieldDescriptor) fieldDefault {
	switch descriptor.Kind() {
	case protoreflect.BytesKind:
		data := descriptor.Default().Bytes()
		printable := strings.Join(
			slicesext.Map(data, func(b byte) string { return fmt.Sprintf("0x%X", b) }),
			",")
		return fieldDefault{
			// cannot compare slices, so we convert []byte to string
			comparable: string(data),
			printable:  "[" + printable + "]",
		}
	case protoreflect.StringKind:
		return fieldDefault{
			comparable: descriptor.Default().String(),
			printable:  fmt.Sprintf("%q", descriptor.Default().String()),
		}
	case protoreflect.EnumKind:
		// Ideally, we'd use descriptor.DefaultEnumValue(). But that returns
		// nil when the default is not explicitly set :(
		enumVal := descriptor.Default().Enum()
		enumDescriptor := descriptor.Enum()
		var printable string
		if enumValDescriptor := enumDescriptor.Values().ByNumber(enumVal); enumValDescriptor != nil {
			printable = fmt.Sprintf("%s.%s", enumDescriptor.Name(), enumValDescriptor.Name())
		} else {
			// should not be possible
			printable = fmt.Sprintf("%s.%d(?)", enumDescriptor.Name(), enumVal)
		}
		return fieldDefault{
			comparable: int32(enumVal),
			printable:  printable,
		}
	default:
		// All other kinds (bool, numbers) compare and print just fine.
		// (We're not considering list, map, or message types since such
		// fields cannot have default values.)
		return fieldDefault{
			comparable: descriptor.Default().Interface(),
			printable:  descriptor.Default().Interface(),
		}
	}
}

func defaultsEqual(previous, current fieldDefault) bool {
	// Since the FIELD_SAME_TYPE check will catch type change errors, we try
	// to be lenient about types here. Basically, we can successfully compare
	// defaults between strings and bytes and then between all other numeric
	// types (including bool and enum). But changing a field from string to
	// number (or vice versa) will trigger an issue about default value change.
	_, previousIsString := previous.comparable.(string)
	_, currentIsString := current.comparable.(string)
	if previousIsString || currentIsString {
		return previous.comparable == current.comparable
	}
	// If neither are strings, we know they are both bools or numeric types.

	// If the type changed from float to double or vice versa, then
	// the big.Float approach below can report a difference due to
	// the extra precision of float64 meaning the value is not
	// *precisely* equal to the float32 form. So for these changes,
	// we convert both to float32 to do the comparison.
	switch previousFloat := previous.comparable.(type) {
	case float32:
		if currentFloat, ok := current.comparable.(float64); ok {
			return (math.IsNaN(float64(previousFloat)) && math.IsNaN(currentFloat)) ||
				previousFloat == float32(currentFloat)
		}
	case float64:
		if currentFloat, ok := current.comparable.(float32); ok {
			return (math.IsNaN(previousFloat) && math.IsNaN(float64(currentFloat))) ||
				float32(previousFloat) == currentFloat
		}
	}

	// To compare values without overflow (and without an inordinate number
	// of switches), we convert them to a common numeric format and compare that.
	// We use *big.Float since it can represent the full range of values
	// for float64, int64, or uint64 without any loss of precision.
	previousVal, previousNaN := asBigFloat(previous.comparable)
	currentVal, currentNaN := asBigFloat(current.comparable)
	if previousNaN && currentNaN {
		return true
	} else if previousNaN != currentNaN {
		return false
	}
	if previousVal == nil || currentVal == nil {
		// should not be possible; but just in case, don't panic
		return previous.comparable == current.comparable
	}
	return previousVal.Cmp(currentVal) == 0
}

func asBigFloat(val any) (result *big.Float, isNaN bool) {
	switch val := val.(type) {
	case bool:
		if val {
			return big.NewFloat(1), false
		}
		return big.NewFloat(0), false
	case int32:
		var float big.Float
		float.SetInt64(int64(val))
		return &float, false
	case int64:
		var float big.Float
		float.SetInt64(val)
		return &float, false
	case uint32:
		var float big.Float
		float.SetUint64(uint64(val))
		return &float, false
	case uint64:
		var float big.Float
		float.SetUint64(val)
		return &float, false
	case float32:
		if math.IsNaN(float64(val)) {
			return nil, true
		}
		return big.NewFloat(float64(val)), false
	case float64:
		if math.IsNaN(val) {
			return nil, true
		}
		return big.NewFloat(val), false
	default:
		// should never happen...
		return nil, false
	}
}

func canHaveDefault(descriptor protoreflect.FieldDescriptor) bool {
	return !descriptor.IsList() && !descriptor.IsMap() && descriptor.Message() == nil
}
