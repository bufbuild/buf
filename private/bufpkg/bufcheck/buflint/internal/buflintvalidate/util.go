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
	"reflect"

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

func validateNumberField[T any](
	m *validateField,
	in, notIn int,
	constIn, lessThanIn, lessThanEqualIn, greaterThanIn, greaterThanEqualIn T,
) {
	m.checkIns(in, notIn)

	constant := reflect.ValueOf(constIn)
	lessThan, lessThanEqual := reflect.ValueOf(lessThanIn), reflect.ValueOf(lessThanEqualIn)
	greaterThan, greaterThanEqual := reflect.ValueOf(greaterThanIn), reflect.ValueOf(greaterThanEqualIn)

	m.assertf(constant.IsNil() ||
		in == 0 && notIn == 0 &&
			lessThan.IsNil() && lessThanEqual.IsNil() &&
			greaterThan.IsNil() && greaterThanEqual.IsNil(),
		"const can be the only rule on a field",
	)

	m.assertf(in == 0 ||
		lessThan.IsNil() && lessThanEqual.IsNil() &&
			greaterThan.IsNil() && greaterThanEqual.IsNil(),
		"cannot have both in and range constraint rules on the same field",
	)

	if !lessThan.IsNil() {
		m.assertf(greaterThan.IsNil() || !reflect.DeepEqual(lessThanIn, greaterThanIn),
			"cannot have equal gt and lt rules on the same field")
		m.assertf(greaterThanEqual.IsNil() || !reflect.DeepEqual(lessThanIn, greaterThanEqualIn),
			"cannot have equal gte and lt rules on the same field")
	} else if !lessThanEqual.IsNil() {
		m.assertf(greaterThan.IsNil() || !reflect.DeepEqual(lessThanEqualIn, greaterThanIn),
			"cannot have equal gt and lte rules on the same field")
		m.assertf(greaterThanEqual.IsNil() || !reflect.DeepEqual(lessThanEqualIn, greaterThanEqualIn),
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
