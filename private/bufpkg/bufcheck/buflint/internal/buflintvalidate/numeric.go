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

var fieldNumberToCheckNumberRulesFunc = map[int32]func(*adder, int32, protoreflect.Message) error{
	floatRulesFieldNumber:    checkNumberRules[float32],
	doubleRulesFieldNumber:   checkNumberRules[float64],
	int32RulesFieldNumber:    checkNumberRules[int32],
	int64RulesFieldNumber:    checkNumberRules[int64],
	uInt32RulesFieldNumber:   checkNumberRules[uint32],
	uInt64RulesFieldNumber:   checkNumberRules[uint64],
	sInt32RulesFieldNumber:   checkNumberRules[int32],
	sInt64RulesFieldNumber:   checkNumberRules[int64],
	fixed32RulesFieldNumber:  checkNumberRules[uint32],
	fixed64RulesFieldNumber:  checkNumberRules[uint64],
	sFixed32RulesFieldNumber: checkNumberRules[int32],
	sFixed64RulesFieldNumber: checkNumberRules[int64],
}

func checkNumberRules[
	T int32 | int64 | uint32 | uint64 | float32 | float64,
](
	adder *adder,
	numberRuleFieldNumber int32,
	ruleMessage protoreflect.Message,
) error {
	return checkNumericRules[T](
		adder,
		numberRuleFieldNumber,
		ruleMessage,
		getNumericPointerFromValue[T],
		compareNumber[T],
	)
}

func checkNumericRules[
	T int32 | int64 | uint32 | uint64 | float32 | float64 | timestamppb.Timestamp | durationpb.Duration,
](
	adder *adder,
	ruleNumber int32,
	message protoreflect.Message,
	// convertFunc returns the converted value, a file annotation string and an error.
	convertFunc func(protoreflect.Value) (*T, string, error),
	// compareFunc returns a positive value if the first argument is bigger,
	// a negative value if the second argument is bigger or 0 if they are equal.
	compareFunc func(*T, *T) float64,
) error {
	var constant, lowerBound, gt, gte, upperBound, lt, lte *T
	var lowerBoundName, upperBoundName string
	var in, notIn []*T
	var fieldCount int
	var constFieldNumber, inFieldNumber, notInFieldNumber, lowerBoundFieldNumber, upperBoundFieldNumber int32
	var err error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldCount++
		var convertErrorMessage string
		switch fieldName := string(field.Name()); fieldName {
		case "const":
			constFieldNumber = int32(field.Number())
			constant, convertErrorMessage, err = convertFunc(value)
		case "gt":
			gt, convertErrorMessage, err = convertFunc(value)
			lowerBound = gt
			lowerBoundName = fieldName
			lowerBoundFieldNumber = int32(field.Number())
		case "gte":
			gte, convertErrorMessage, err = convertFunc(value)
			lowerBound = gte
			lowerBoundName = fieldName
			lowerBoundFieldNumber = int32(field.Number())
		case "lt":
			lt, convertErrorMessage, err = convertFunc(value)
			upperBound = lt
			upperBoundName = fieldName
			upperBoundFieldNumber = int32(field.Number())
		case "lte":
			lte, convertErrorMessage, err = convertFunc(value)
			upperBound = lte
			upperBoundName = fieldName
			upperBoundFieldNumber = int32(field.Number())
		case "in":
			inFieldNumber = int32(field.Number())
			for i := 0; i < value.List().Len(); i++ {
				var converted *T
				converted, convertErrorMessage, err = convertFunc(value.List().Get(i))
				if converted != nil {
					in = append(in, converted)
				}
			}
		case "not_in":
			notInFieldNumber = int32(field.Number())
			for i := 0; i < value.List().Len(); i++ {
				var converted *T
				converted, convertErrorMessage, err = convertFunc(value.List().Get(i))
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
	if err != nil {
		return err
	}
	if constant != nil && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ruleNumber, constFieldNumber},
			"const should be the only rule when specified",
		)
	}
	if len(in) > 0 && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ruleNumber, inFieldNumber},
			"in should be the only rule when specified",
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
		return nil
	}
	if gte != nil && lte != nil && compareFunc(upperBound, lowerBound) == 0 {
		adder.addForPathsf(
			[][]int32{
				{ruleNumber, lowerBoundFieldNumber},
				{ruleNumber, upperBoundFieldNumber},
			},
			"lte and gte have the same value, consider using const",
		)
		return nil
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
	return nil
}

func getNumericPointerFromValue[
	T int32 | int64 | uint32 | uint64 | float32 | float64,
](value protoreflect.Value) (*T, string, error) {
	number, ok := value.Interface().(T)
	if !ok {
		return nil, "", fmt.Errorf("unable to cast value to type %T", number)
	}
	return &number, "", nil
}

func getTimestampFromValue(value protoreflect.Value) (*timestamppb.Timestamp, string, error) {
	bytes, err := proto.Marshal(value.Message().Interface())
	if err != nil {
		return nil, "", err
	}
	timestamp := &timestamppb.Timestamp{}
	err = proto.Unmarshal(bytes, timestamp)
	if err != nil {
		return nil, "", err
	}
	if !timestamp.IsValid() {
		return nil, fmt.Sprintf("%v is not a valid timestamp", timestamp), nil
	}
	return timestamp, "", nil
}

func getDurationFromValue(value protoreflect.Value) (*durationpb.Duration, string, error) {
	bytes, err := proto.Marshal(value.Message().Interface())
	if err != nil {
		return nil, "", err
	}
	duration := &durationpb.Duration{}
	err = proto.Unmarshal(bytes, duration)
	if err != nil {
		return nil, "", err
	}
	if !duration.IsValid() {
		return nil, fmt.Sprintf("%v is an invalid duration", duration), nil
	}
	return duration, "", nil
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
