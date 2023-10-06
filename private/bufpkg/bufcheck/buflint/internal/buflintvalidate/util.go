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
	"os"
	"time"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protosource"
)

var (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.Int32Rules and
	// and all other <number type>Rules have the same set of tags.
	defaultNumericTagSet = numericTagSet{
		constant: 1,
		lt:       2,
		lte:      3,
		gt:       4,
		gte:      5,
		in:       6,
		notIn:    7,
	}
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.DurationRules
	durationNumericTagSet = numericTagSet{
		constant: 2,
		lt:       3,
		lte:      4,
		gt:       5,
		gte:      6,
		in:       7,
		notIn:    8,
	}
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.TimestampRules
	timestampNumericTagSet = numericTagSet{
		constant: 2,
		lt:       3,
		lte:      4,
		gt:       5,
		gte:      6,
	}
)

type numericTagSet struct {
	constant int32
	lt       int32
	lte      int32
	gt       int32
	gte      int32
	in       int32
	notIn    int32
}

type numericRules[N comparable] interface {
	GetGt() N
	GetGte() N
	GetLt() N
	GetLte() N
}

type numericRange[T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration] struct {
	lowerBound            *T
	isLowerBoundInclusive bool
	upperBound            *T
	isUpperBoundInclusive bool
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
	fmt.Fprintf(os.Stderr, "THIS IS: %s\n", r)
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

func resolveLimits[
	N comparable,
	GT any,
	GTE any,
	LT any,
	LTE any,
](
	rules numericRules[N],
	gtOneOf any,
	ltOneOf any,
) (gt, gte, lt, lte *N) {
	switch gtOneOf.(type) {
	case GT:
		n := rules.GetGt()
		gt = &n
	case GTE:
		n := rules.GetGte()
		gte = &n
	}
	switch ltOneOf.(type) {
	case LT:
		n := rules.GetLt()
		lt = &n
	case LTE:
		n := rules.GetLte()
		lte = &n
	}
	return
}

func validateNumberField[T int32 | int64 | uint32 | uint64 | float32 | float64 | time.Duration](
	m *validateField,
	ruleTag int32,
	tagSet numericTagSet,
	in, notIn int,
	constant, greaterThan, greaterThanEqual, lessThan, lessThanEqual *T,
) {
	m.checkIns(in, notIn)
	numberRange := newNumericRange(greaterThan, greaterThanEqual, lessThan, lessThanEqual)
	if constant != nil && (in != 0 || notIn != 0 || numberRange.isDefined()) {
		m.add(
			m.field,
			m.field.OptionExtensionLocation(validate.E_Field, ruleTag, tagSet.constant),
			nil,
			"all other rules are ignored when const is specified on a field",
		)
	}
	if in != 0 && numberRange.isDefined() {
		m.add(
			m.field,
			m.field.OptionExtensionLocation(validate.E_Field, ruleTag, tagSet.in),
			nil,
			"cannot have both in and range constraint rules on the same field",
		)
	}
	if !numberRange.isValid() {
		m.add(
			m.field,
			m.field.OptionExtensionLocation(validate.E_Field, ruleTag),
			nil,
			"%v is not a valid range",
			numberRange,
		)
	}
	if numberRange.allowsSingleValue() {
		m.add(
			m.field,
			m.field.OptionExtensionLocation(validate.E_Field, ruleTag),
			nil,
			"use const instead of equal lte and gte values for range %v",
			numberRange,
		)
	}
}

func embed(f protosource.Field, files ...protosource.File) protosource.Message {
	fullNameToMessage, err := protosource.FullNameToMessage(files...)
	if err != nil {
		return nil
	}
	out, ok := fullNameToMessage[f.TypeName()]
	if !ok {
		return nil
	}
	return out
}

func getEnum(
	f protosource.Field,
	files ...protosource.File,
) protosource.Enum {
	fullNameToEnum, err := protosource.FullNameToEnum(files...)
	if err != nil {
		return nil
	}
	out, ok := fullNameToEnum[f.TypeName()]
	if !ok {
		return nil
	}
	return out
}
