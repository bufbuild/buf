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
	"time"

	"github.com/bufbuild/buf/private/pkg/protosource"
)

type numericRules[N comparable] interface {
	GetGt() N
	GetGte() N
	GetLt() N
	GetLte() N
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
	in, notIn int,
	constant, greaterThan, greaterThanEqual, lessThan, lessThanEqual *T,
) {
	m.checkIns(in, notIn)

	m.assertf(constant == nil ||
		in == 0 && notIn == 0 &&
			lessThan == nil && lessThanEqual == nil &&
			greaterThan == nil && greaterThanEqual == nil,
		"const can be the only rule on a field",
	)

	m.assertf(in == 0 ||
		lessThan == nil && lessThanEqual == nil &&
			greaterThan == nil && greaterThanEqual == nil,
		"cannot have both in and range constraint rules on the same field",
	)

	if !(lessThan == nil) {
		m.assertf(greaterThan == nil || *lessThan != *greaterThan,
			"cannot have equal gt and lt rules on the same field")
		m.assertf(greaterThanEqual == nil || *lessThan != *greaterThanEqual,
			"cannot have equal gte and lt rules on the same field")
	} else if !(lessThanEqual == nil) {
		m.assertf(greaterThan == nil || *lessThanEqual != *greaterThan,
			"cannot have equal gt and lte rules on the same field")
		m.assertf(greaterThanEqual == nil || *lessThanEqual != *greaterThanEqual,
			"use const instead of equal lte and gte rules")
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
