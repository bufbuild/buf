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
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func validateNumberRulesMessage(
	adder *adder,
	numberRuleFieldNumber int32,
	numberRuleMessage protoreflect.Message,
) {
	validateFunc := numberRulesFieldNumberToValidateFunc[numberRuleFieldNumber]
	validateFunc(adder, numberRuleFieldNumber, numberRuleMessage)
}

func validateNumRule[T int32 | int64 | uint32 | uint64 | float32 | float64](
	adder *adder,
	numberRuleFieldNumber int32,
	ruleMessage protoreflect.Message,
) {
	validateNumericRule[T](
		adder,
		numberRuleFieldNumber,
		ruleMessage,
		getNumericPointer[T],
		compareNumber[T],
	)
}

func validateNumericRule[
	T int32 | int64 | uint32 | uint64 | float32 | float64 | copiableTime,
](
	adder *adder,
	ruleNumber int32,
	message protoreflect.Message,
	convertFunc func(protoreflect.Value) (*T, string),
	compareFunc func(T, T) float64,
) {
	var constant, lowerBound, gt, gte, upperBound, lt, lte *T
	var lowerBoundName, upperBoundName string
	var in, notIn []T
	var fieldCount int
	// TODO: set field numbers during the loop
	// TODO: make convertFunc return a file annotation as well
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldCount++
		var convertErrorMessage string
		switch fieldName := string(field.Name()); fieldName {
		case "const":
			constant, convertErrorMessage = convertFunc(value)
		case "gt":
			gt, convertErrorMessage = convertFunc(value)
			lowerBound = gt
			lowerBoundName = fieldName
		case "gte":
			gte, convertErrorMessage = convertFunc(value)
			lowerBound = gte
			lowerBoundName = fieldName
		case "lt":
			lt, convertErrorMessage = convertFunc(value)
			upperBound = lt
			upperBoundName = fieldName
		case "lte":
			lte, convertErrorMessage = convertFunc(value)
			upperBound = lte
			upperBoundName = fieldName
		case "in":
			for i := 0; i < value.List().Len(); i++ {
				var converted *T
				converted, convertErrorMessage = convertFunc(value.List().Get(i))
				if converted != nil {
					in = append(in, *converted)
				}
			}
		case "not_in":
			for i := 0; i < value.List().Len(); i++ {
				var converted *T
				converted, convertErrorMessage = convertFunc(value.List().Get(i))
				if converted != nil {
					notIn = append(notIn, *converted)
				}
			}
		}
		if convertErrorMessage != "" {
			adder.addForPath(
				[]int32{ruleNumber, int32(field.Number())},
				convertErrorMessage,
			)
		}
		return true
	})
	if constant != nil && fieldCount > 1 {
		adder.addForPath(
			[]int32{ruleNumber},
			"all other rules are ignored when const is specified on a field",
		)
	}
	if len(in) > 0 && fieldCount > 1 {
		adder.addForPath(
			[]int32{ruleNumber},
			"in should be the only rule when defined",
		)
	}
	for _, bannedValue := range notIn {
		var failedChecks []string
		if gt != nil && compareFunc(bannedValue, *gt) <= 0 {
			failedChecks = append(failedChecks, "gt")
		}
		if gte != nil && compareFunc(bannedValue, *gte) < 0 {
			failedChecks = append(failedChecks, "gte")
		}
		if lt != nil && compareFunc(bannedValue, *lt) >= 0 {
			failedChecks = append(failedChecks, "lt")
		}
		if lte != nil && compareFunc(bannedValue, *lte) > 0 {
			failedChecks = append(failedChecks, "lte")
		}
		if len(failedChecks) > 0 {
			adder.addForPath(
				[]int32{ruleNumber},
				"%v is already rejected by %s and does not need to be in not_in",
				bannedValue,
				// TODO: string util?
				strings.Join(failedChecks, " and "),
			)
		}
	}
	if lowerBound == nil || upperBound == nil {
		return
	}
	if gte != nil && lte != nil && *lowerBound == *upperBound {
		adder.addForPath(
			[]int32{ruleNumber},
			"lte and gte have the same value, consider using const",
		)
		return
	}
	if compareFunc(*upperBound, *lowerBound) <= 0 {
		adder.addForPath(
			[]int32{ruleNumber},
			"%s should be greater than %s",
			upperBoundName,
			lowerBoundName,
		)
	}
}

type validateNumberRuleFunc func(*adder, int32, protoreflect.Message)

var numberRulesFieldNumberToValidateFunc = map[int32]validateNumberRuleFunc{
	floatRulesFieldNumber:    validateNumRule[float32],
	doubleRulesFieldNumber:   validateNumRule[float64],
	int32RulesFieldNumber:    validateNumRule[int32],
	int64RulesFieldNumber:    validateNumRule[int64],
	uInt32RulesFieldNumber:   validateNumRule[uint32],
	uInt64RulesFieldNumber:   validateNumRule[uint64],
	sInt32RulesFieldNumber:   validateNumRule[int32],
	sInt64RulesFieldNumber:   validateNumRule[int64],
	fixed32RulesFieldNumber:  validateNumRule[uint32],
	fixed64RulesFieldNumber:  validateNumRule[uint64],
	sFixed32RulesFieldNumber: validateNumRule[int32],
	sFixed64RulesFieldNumber: validateNumRule[int64],
}

func getNumericPointer[
	T int32 | int64 | uint32 | uint64 | float32 | float64,
](value protoreflect.Value) (*T, string) {
	pointer := value.Interface().(T)
	return &pointer, ""
}

func compareNumber[T int32 | int64 | uint32 | uint64 | float32 | float64](a T, b T) float64 {
	return float64(a - b)
}

type copiableTime struct {
	seconds int64
	nanos   int32
}

func compareTime(t1 copiableTime, t2 copiableTime) float64 {
	if t1.seconds > t2.seconds {
		return 1
	}
	if t1.seconds < t2.seconds {
		return -1
	}
	return float64(t1.nanos - t2.nanos)
}
