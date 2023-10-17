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

package buflintvalidate

import (
	"fmt"

	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var numberRulesFieldNumberToValidateFunc = map[int32]func(*adder, int32, protoreflect.Message){
	floatRulesFieldNumber:    validateNumberRule[float32],
	doubleRulesFieldNumber:   validateNumberRule[float64],
	int32RulesFieldNumber:    validateNumberRule[int32],
	int64RulesFieldNumber:    validateNumberRule[int64],
	uInt32RulesFieldNumber:   validateNumberRule[uint32],
	uInt64RulesFieldNumber:   validateNumberRule[uint64],
	sInt32RulesFieldNumber:   validateNumberRule[int32],
	sInt64RulesFieldNumber:   validateNumberRule[int64],
	fixed32RulesFieldNumber:  validateNumberRule[uint32],
	fixed64RulesFieldNumber:  validateNumberRule[uint64],
	sFixed32RulesFieldNumber: validateNumberRule[int32],
	sFixed64RulesFieldNumber: validateNumberRule[int64],
}

func validateNumberRule[
	T int32 | int64 | uint32 | uint64 | float32 | float64,
](
	adder *adder,
	numberRuleFieldNumber int32,
	ruleMessage protoreflect.Message,
) {
	validateNumericRule[T](
		adder,
		numberRuleFieldNumber,
		ruleMessage,
		getNumericPointerFromValue[T],
		compareNumber[T],
	)
}

func validateNumericRule[
	T int32 | int64 | uint32 | uint64 | float32 | float64 | timestamppb.Timestamp | durationpb.Duration,
](
	adder *adder,
	ruleNumber int32,
	message protoreflect.Message,
	// These two functions must take pointers because of the generated types.
	convertFunc func(protoreflect.Value) (*T, string),
	compareFunc func(*T, *T) float64,
) {
	var constant, lowerBound, gt, gte, upperBound, lt, lte *T
	var lowerBoundName, upperBoundName string
	var in, notIn []*T
	var fieldCount int
	var constFieldNumber, inFieldNumber, notInFieldNumber, lowerBoundFieldNumber, upperBoundFieldNumber int32
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldCount++
		var convertErrorMessage string
		switch fieldName := string(field.Name()); fieldName {
		case "const":
			constFieldNumber = int32(field.Number())
			constant, convertErrorMessage = convertFunc(value)
		case "gt":
			gt, convertErrorMessage = convertFunc(value)
			lowerBound = gt
			lowerBoundName = fieldName
			lowerBoundFieldNumber = int32(field.Number())
		case "gte":
			gte, convertErrorMessage = convertFunc(value)
			lowerBound = gte
			lowerBoundName = fieldName
			lowerBoundFieldNumber = int32(field.Number())
		case "lt":
			lt, convertErrorMessage = convertFunc(value)
			upperBound = lt
			upperBoundName = fieldName
			upperBoundFieldNumber = int32(field.Number())
		case "lte":
			lte, convertErrorMessage = convertFunc(value)
			upperBound = lte
			upperBoundName = fieldName
			upperBoundFieldNumber = int32(field.Number())
		case "in":
			inFieldNumber = int32(field.Number())
			for i := 0; i < value.List().Len(); i++ {
				var converted *T
				converted, convertErrorMessage = convertFunc(value.List().Get(i))
				if converted != nil {
					in = append(in, converted)
				}
			}
		case "not_in":
			notInFieldNumber = int32(field.Number())
			for i := 0; i < value.List().Len(); i++ {
				var converted *T
				converted, convertErrorMessage = convertFunc(value.List().Get(i))
				if converted != nil {
					notIn = append(notIn, converted)
				}
			}
		}
		if convertErrorMessage != "" {
			adder.addForPathf(
				[]int32{ruleNumber, int32(field.Number())},
				convertErrorMessage,
			)
		}
		return true
	})
	if constant != nil && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ruleNumber, constFieldNumber},
			"all other rules are redundant when const is specified on a field",
		)
	}
	if len(in) > 0 && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ruleNumber, inFieldNumber},
			"in should be the only rule when defined",
		)
	}
	for _, bannedValue := range notIn {
		var failedChecks []string
		if gt != nil && compareFunc(bannedValue, gt) <= 0 {
			failedChecks = append(failedChecks, "gt")
		}
		if gte != nil && compareFunc(bannedValue, gte) < 0 {
			failedChecks = append(failedChecks, "gte")
		}
		if lt != nil && compareFunc(bannedValue, lt) >= 0 {
			failedChecks = append(failedChecks, "lt")
		}
		if lte != nil && compareFunc(bannedValue, lte) > 0 {
			failedChecks = append(failedChecks, "lte")
		}
		if len(failedChecks) > 0 {
			adder.addForPathf(
				[]int32{ruleNumber, notInFieldNumber},
				"%v is already rejected by %s and does not need to be in not_in",
				bannedValue,
				stringutil.SliceToHumanString(failedChecks),
			)
		}
	}
	if lowerBound == nil || upperBound == nil {
		return
	}
	if gte != nil && lte != nil && compareFunc(upperBound, lowerBound) == 0 {
		adder.addForPathsf(
			[][]int32{
				{ruleNumber, lowerBoundFieldNumber},
				{ruleNumber, upperBoundFieldNumber},
			},
			"lte and gte have the same value, consider using const",
		)
		return
	}
	if compareFunc(upperBound, lowerBound) <= 0 {
		adder.addForPathsf(
			[][]int32{
				{ruleNumber, lowerBoundFieldNumber},
				{ruleNumber, upperBoundFieldNumber},
			},
			"%s should be greater than %s",
			upperBoundName,
			lowerBoundName,
		)
	}
}

func getNumericPointerFromValue[
	T int32 | int64 | uint32 | uint64 | float32 | float64,
](value protoreflect.Value) (*T, string) {
	pointer, _ := value.Interface().(T)
	return &pointer, ""
}

func getTimestampFromValue(value protoreflect.Value) (*timestamppb.Timestamp, string) {
	// TODO: what if this errors?
	bytes, _ := proto.Marshal(value.Message().Interface())
	timestamp := &timestamppb.Timestamp{}
	_ = proto.Unmarshal(bytes, timestamp)
	if !timestamp.IsValid() {
		return nil, fmt.Sprintf("%v is not a valid timestamp", timestamp)
	}
	return timestamp, ""
}

func getDurationFromValue(value protoreflect.Value) (*durationpb.Duration, string) {
	// TODO: what if this errors?
	bytes, _ := proto.Marshal(value.Message().Interface())
	duration := &durationpb.Duration{}
	_ = proto.Unmarshal(bytes, duration)
	if !duration.IsValid() {
		return nil, fmt.Sprintf("%v is an invalid duration", duration)
	}
	return duration, ""
}

func compareNumber[T int32 | int64 | uint32 | uint64 | float32 | float64](a *T, b *T) float64 {
	return float64(*a - *b)
}

func compareTimestamp(t1 *timestamppb.Timestamp, t2 *timestamppb.Timestamp) float64 {
	if t1.Seconds > t2.Seconds {
		return 1
	}
	if t1.Seconds < t2.Seconds {
		return -1
	}
	return float64(t1.Nanos - t2.Nanos)
}

func compareDuration(d1 *durationpb.Duration, d2 *durationpb.Duration) float64 {
	if d1.Seconds > d2.Seconds {
		return 1
	}
	if d1.Seconds < d2.Seconds {
		return -1
	}
	return float64(d1.Nanos - d2.Nanos)
}
