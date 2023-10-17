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
	"strings"
	"time"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// commonNumericFieldNumberSet is the set of field numbers for fields const, lt, lte,
// gt, gte, in and notIn, common to <Numeric Type>Rules, such as Int32Rules and TimestampRules.
type commonNumericFieldNumberSet struct {
	constant int32
	lt       int32
	lte      int32
	gt       int32
	gte      int32
	in       int32
	notIn    int32
}

var (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.Int32Rules and
	// and all other <number type>Rules have the same set field numbers for the common fields.
	defaultFieldNumberSet = commonNumericFieldNumberSet{
		constant: 1,
		lt:       2,
		lte:      3,
		gt:       4,
		gte:      5,
		in:       6,
		notIn:    7,
	}
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.DurationRules
	durationFieldNumberSet = commonNumericFieldNumberSet{
		constant: 2,
		lt:       3,
		lte:      4,
		gt:       5,
		gte:      6,
		in:       7,
		notIn:    8,
	}
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.TimestampRules
	timestampFieldNumberSet = commonNumericFieldNumberSet{
		constant: 2,
		lt:       3,
		lte:      4,
		gt:       5,
		gte:      6,
	}
)

type numericCommonRule[T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration] struct {
	constant   *T
	valueRange numericRange[T]
	in         []T
	notIn      []T
}

type numericRange[T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration] struct {
	lowerBound            *T
	isLowerBoundInclusive bool
	upperBound            *T
	isUpperBoundInclusive bool
}

func validateNumericRule[T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration](
	adder *adder,
	m *validateField,
	ruleTag int32,
	tagSet commonNumericFieldNumberSet,
	ruleMessage protoreflect.Message,
) {
	rules := normalizeConstraints[T](ruleMessage)
	validateCommonNumericRule(
		adder,
		m,
		ruleTag,
		tagSet,
		rules,
	)
}

func validateTimeRule[
	T durationpb.Duration | timestamppb.Timestamp,
	U copiableTime,
](
	adder *adder,
	validateField *validateField,
	field protosource.Field,
	ruleNumber int32,
	message protoreflect.Message,
	convertFunc func(protoreflect.Value) *U,
	compareFunc func(U, U) int,
) {
	var constant, lowerBound, gt, gte, upperBound, lt, lte *U
	var lowerBoundName, upperBoundName string
	var in, notIn []U
	var fieldCount int
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldCount++
		switch fieldName := string(field.Name()); fieldName {
		case "const":
			constant = convertFunc(value)
		case "gt":
			gt = convertFunc(value)
			lowerBound = gt
			lowerBoundName = fieldName
		case "gte":
			gte = convertFunc(value)
			lowerBound = gte
			lowerBoundName = fieldName
		case "lt":
			lt = convertFunc(value)
			upperBound = lt
			upperBoundName = fieldName
		case "lte":
			lte = convertFunc(value)
			upperBound = lte
			upperBoundName = fieldName
		case "in":
			for i := 0; i < value.List().Len(); i++ {
				u := convertFunc(value.List().Get(i))
				if u != nil {
					in = append(in, *u)
				}
			}
		case "not_in":
			for i := 0; i < value.List().Len(); i++ {
				u := convertFunc(value.List().Get(i))
				if u != nil {
					notIn = append(notIn, *u)
				}
			}
		}
		return true
	})
	if constant != nil && fieldCount > 1 {
		validateField.add(
			validateField.field,
			// TODO: tagset constant
			validateField.field.OptionExtensionLocation(validate.E_Field, ruleNumber), // TODO: add path
			nil,
			"all other rules are ignored when const is specified on a field",
		)
	}
	checkIns(adder, len(in), len(notIn))
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
			validateField.add(
				validateField.field,
				validateField.field.OptionExtensionLocation(validate.E_Field, ruleNumber), // TODO: add path
				nil,
				"%v is already rejected by %s and does not need to be in not_in",
				bannedValue,
				strings.Join(failedChecks, " and "),
			)
		}
	}
	if gte != nil && lte != nil && *lowerBound == *upperBound {
		validateField.add(
			validateField.field,
			validateField.field.OptionExtensionLocation(validate.E_Field, ruleNumber), // TODO: add path
			nil,
			"lte and gte have the same value, consider using const",
		)
		return
	}
	if lowerBound == nil || upperBound == nil {
		return
	}
	if compareFunc(*upperBound, *lowerBound) <= 0 {
		validateField.add(
			validateField.field,
			validateField.field.OptionExtensionLocation(validate.E_Field, ruleNumber), // TODO: add path
			nil,
			"%s should be greater than %s",
			upperBoundName,
			lowerBoundName,
		)
	}
}

type validateNumberRuleFunc func(*adder, *validateField, int32, commonNumericFieldNumberSet, protoreflect.Message)

func validateNumberRulesMessage(
	adder *adder,
	validateField *validateField,
	field protosource.Field,
	ruleNumber int32,
	numberRuleMessage protoreflect.Message,
) {
	validateFunc := tagToValidateFunc[ruleNumber]
	validateFunc(adder, validateField, ruleNumber, defaultFieldNumberSet, numberRuleMessage)
}

var tagToValidateFunc = map[int32]validateNumberRuleFunc{
	floatRulesFieldNumber:    validateNumericRule[float32],
	doubleRulesFieldNumber:   validateNumericRule[float64],
	int32RulesFieldNumber:    validateNumericRule[int32],
	int64RulesFieldNumber:    validateNumericRule[int64],
	uInt32RulesFieldNumber:   validateNumericRule[uint32],
	uInt64RulesFieldNumber:   validateNumericRule[uint64],
	sInt32RulesFieldNumber:   validateNumericRule[int32],
	sInt64RulesFieldNumber:   validateNumericRule[int64],
	fixed32RulesFieldNumber:  validateNumericRule[uint32],
	fixed64RulesFieldNumber:  validateNumericRule[uint64],
	sFixed32RulesFieldNumber: validateNumericRule[int32],
	sFixed64RulesFieldNumber: validateNumericRule[int64],
}

func validateCommonNumericRule[T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration](
	adder *adder,
	m *validateField,
	ruleTag int32,
	tagSet commonNumericFieldNumberSet,
	rules *numericCommonRule[T],
) {
	checkIns(adder, len(rules.in), len(rules.notIn))
	if rules.constant != nil && (len(rules.in) != 0 || len(rules.notIn) != 0 || rules.valueRange.isDefined()) {
		m.add(
			m.field,
			m.field.OptionExtensionLocation(validate.E_Field, ruleTag, tagSet.constant),
			nil,
			"all other rules are ignored when const is specified on a field",
		)
	}
	if len(rules.in) != 0 && rules.valueRange.isDefined() {
		m.add(
			m.field,
			m.field.OptionExtensionLocation(validate.E_Field, ruleTag, tagSet.in),
			nil,
			"cannot have both in and range constraint rules on the same field",
		)
	}
	if !rules.valueRange.isValid() {
		m.add(
			m.field,
			m.field.OptionExtensionLocation(validate.E_Field, ruleTag),
			nil,
			"%v is not a valid range",
			rules.valueRange,
		)
	}
	if rules.valueRange.allowsSingleValue() {
		m.add(
			m.field,
			m.field.OptionExtensionLocation(validate.E_Field, ruleTag),
			nil,
			"use const instead of equal lte and gte values for range %v",
			rules.valueRange,
		)
	}
}

func normalizeConstraints[
	T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration,
](message protoreflect.Message) *numericCommonRule[T] {
	var constant, gt, gte, lt, lte *T
	var in, notIn []T
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch string(field.Name()) {
		case "const":
			constant = getNumericPointer[T](value.Interface())
		case "gt":
			gt = getNumericPointer[T](value.Interface())
		case "gte":
			gte = getNumericPointer[T](value.Interface())
		case "lt":
			lt = getNumericPointer[T](value.Interface())
		case "lte":
			lte = getNumericPointer[T](value.Interface())
		case "in":
			for i := 0; i < value.List().Len(); i++ {
				in = append(in, value.List().Get(i).Interface().(T))
			}
		case "not_in":
			for i := 0; i < value.List().Len(); i++ {
				notIn = append(notIn, value.List().Get(i).Interface().(T))
			}
		}
		return true
	})
	return &numericCommonRule[T]{
		constant:   constant,
		in:         in,
		notIn:      notIn,
		valueRange: *newNumericRange[T](gt, gte, lt, lte),
	}
}

func getNumericPointer[
	T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration,
](value interface{}) *T {
	pointer := value.(T)
	return &pointer
}

func newNumericRange[T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration](
	gt, gte, lt, lte *T,
) *numericRange[T] {
	numericRange := &numericRange[T]{
		lowerBound: gt,
		upperBound: lt,
	}
	if gte != nil {
		numericRange.lowerBound = gte
		numericRange.isLowerBoundInclusive = true
	}
	if lte != nil {
		numericRange.upperBound = lte
		numericRange.isUpperBoundInclusive = true
	}
	return numericRange
}

func (r *numericRange[T]) isDefined() bool {
	return r.lowerBound != nil || r.upperBound != nil
}

func (r *numericRange[T]) allowsSingleValue() bool {
	return r.lowerBound != nil && r.upperBound != nil && *r.lowerBound == *r.upperBound && r.isLowerBoundInclusive && r.isUpperBoundInclusive
}

func (r *numericRange[T]) isValid() bool {
	if r.lowerBound == nil || r.upperBound == nil {
		return true
	}
	if *r.lowerBound < *r.upperBound {
		return true
	}
	if *r.lowerBound > *r.upperBound {
		return false
	}
	return r.isLowerBoundInclusive && r.isUpperBoundInclusive
}

// func (r *numericRange[T]) contains(number T) bool {
// 	if r.lowerBound != nil {
// 		if number < *r.lowerBound {
// 			return false
// 		}
// 		if number == *r.lowerBound && !r.isLowerBoundInclusive {
// 			return false
// 		}
// 	}
// 	if r.upperBound != nil {
// 		if number > *r.upperBound {
// 			return false
// 		}
// 		if number == *r.upperBound && !r.isUpperBoundInclusive {
// 			return false
// 		}
// 	}
// 	return true
// }

func (r *numericRange[T]) String() string {
	leftDelimiter := "("
	if r.isLowerBoundInclusive {
		leftDelimiter = "["
	}
	// type is any because string(<some float>) is not possible.
	var lowerBound any = "-Infinity"
	if r.lowerBound != nil {
		lowerBound = *r.lowerBound
	}
	var upperBound any = "Infinity"
	if r.upperBound != nil {
		upperBound = *r.upperBound
	}
	rightDelimiter := ")"
	if r.isUpperBoundInclusive {
		rightDelimiter = "]"
	}
	return fmt.Sprintf("%s%v,%v%s", leftDelimiter, lowerBound, upperBound, rightDelimiter)
}

type copiableTime struct {
	seconds int64
	nanos   int32
}
