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

package buflintvalidate

import (
	"fmt"

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
		func(t *T) interface{} { return *t },
	)
}

func checkNumericRules[
	T int32 | int64 | uint32 | uint64 | float32 | float64 | timestamppb.Timestamp | durationpb.Duration,
](
	adder *adder,
	ruleFieldNumber int32,
	ruleMessage protoreflect.Message,
	// convertFunc returns the converted value, a file annotation string and an error.
	convertFunc func(protoreflect.Value) (*T, string, error),
	// equalFunc returns whether two values are equal.
	equalFunc func(*T, *T) bool,
	// formatFunc returns the value suitable for printing with %v.
	formatFunc func(*T) interface{},
) error {
	var fieldCount int
	var constant, lowerBound, upperBound *T
	var isLowerBoundInclusive, isUpperBoundInclusive bool
	var constFieldNumber, lowerBoundFieldNumber, upperBoundFieldNumber int32
	var err error
	ruleMessage.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldCount++
		var convertErrorMessage string
		fieldNumber := int32(field.Number())
		switch fieldName := string(field.Name()); fieldName {
		case "const":
			constFieldNumber = fieldNumber
			constant, convertErrorMessage, err = convertFunc(value)
		case "gt":
			lowerBound, convertErrorMessage, err = convertFunc(value)
			lowerBoundFieldNumber = fieldNumber
		case "gte":
			lowerBound, convertErrorMessage, err = convertFunc(value)
			lowerBoundFieldNumber = fieldNumber
			isLowerBoundInclusive = true
		case "lt":
			upperBound, convertErrorMessage, err = convertFunc(value)
			upperBoundFieldNumber = fieldNumber
		case "lte":
			upperBound, convertErrorMessage, err = convertFunc(value)
			upperBoundFieldNumber = fieldNumber
			isUpperBoundInclusive = true
		}
		if convertErrorMessage != "" {
			adder.addForPathf(
				[]int32{ruleFieldNumber, fieldNumber},
				"Field %q has an invalid %s: %s.",
				adder.fieldName(),
				adder.getFieldRuleName(ruleFieldNumber, int32(field.Number())),
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
			[]int32{ruleFieldNumber, constFieldNumber},
			"Field %q has %s, therefore other rules in %s are not applied and should be removed.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, constFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber),
		)
	}
	if lowerBound == nil || upperBound == nil {
		return nil
	}
	// We do not check which one is larger because in protovalidate, both
	// {lt: 3, gt: 5} and {lt: 5, gt: 3} are valid.
	if !equalFunc(upperBound, lowerBound) {
		return nil
	}
	if isUpperBoundInclusive && isLowerBoundInclusive {
		adder.addForPathsf(
			[][]int32{
				{ruleFieldNumber, lowerBoundFieldNumber},
				{ruleFieldNumber, upperBoundFieldNumber},
			},
			"Field %q has equal %s and %s, use %s.const instead.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, lowerBoundFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber, upperBoundFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber),
		)
		return nil
	}
	adder.addForPathsf(
		[][]int32{
			{ruleFieldNumber, lowerBoundFieldNumber},
			{ruleFieldNumber, upperBoundFieldNumber},
		},
		"Field %q has equal %s and %s. All values are rejected by these checks.",
		adder.fieldName(),
		adder.getFieldRuleName(ruleFieldNumber, lowerBoundFieldNumber),
		adder.getFieldRuleName(ruleFieldNumber, upperBoundFieldNumber),
	)
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
	timestampErr := timestamp.CheckValid()
	if timestampErr == nil {
		return timestamp, "", nil
	}
	return nil, timestampErr.Error(), nil
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
	if durationErrString := checkDuration(duration); durationErrString != "" {
		return nil, durationErrString, nil
	}
	return duration, "", nil
}

func compareNumber[T int32 | int64 | uint32 | uint64 | float32 | float64](a *T, b *T) bool {
	return *a == *b
}

func compareTimestamp(t1 *timestamppb.Timestamp, t2 *timestamppb.Timestamp) bool {
	return t1.Seconds == t2.Seconds && t1.Nanos == t2.Nanos
}

func compareDuration(d1 *durationpb.Duration, d2 *durationpb.Duration) bool {
	return d1.Seconds == d2.Seconds && d1.Nanos == d2.Nanos
}

func checkDuration(duration *durationpb.Duration) string {
	// This is slightly smaller than MaxInt64, 9,223,372,036,854,775,807,
	// but 9,223,372,036,854,775,428 is the maximum value that does not cause a
	// runtime error in protovalidate.
	maxDuration := &durationpb.Duration{
		Seconds: 9223372036,
		Nanos:   854775428,
	}
	minDuration := &durationpb.Duration{
		Seconds: -9223372036,
		Nanos:   -854775428,
	}
	secs := duration.GetSeconds()
	nanos := duration.GetNanos()
	switch {
	case nanos <= -1e9 || nanos >= +1e9:
		return fmt.Sprintf("duration (%v) must have nanos in the range 0 to 999999999", duration)
	case (secs > 0 && nanos < 0) || (secs < 0 && nanos > 0):
		return fmt.Sprintf("duration (%v) has seconds and nanos with different signs", duration)
	case duration.AsDuration() > maxDuration.AsDuration() || duration.AsDuration() < minDuration.AsDuration():
		return fmt.Sprintf("duration (%v) must be in the range %v to %v", duration, minDuration, maxDuration)
	}
	return ""
}
