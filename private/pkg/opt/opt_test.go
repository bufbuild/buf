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

package opt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrimitiveZero(t *testing.T) {
	t.Parallel()
	optional := NewOptional[int](0)
	require.Equal(t, 0, optional.Value)
	require.False(t, optional.Present())
}

func TestPrimitivePresent(t *testing.T) {
	t.Parallel()
	optional := NewOptional[int](1)
	require.Equal(t, 1, optional.Value)
	require.True(t, optional.Present())
}

func TestStructZero(t *testing.T) {
	t.Parallel()
	optional := NewOptional[testStruct](testStruct{})
	require.Equal(t, testStruct{}, optional.Value)
	require.False(t, optional.Present())

	optional = NewOptional[testStruct](testStruct{value: 0})
	require.Equal(t, testStruct{}, optional.Value)
	require.False(t, optional.Present())
}

func TestStructPresent(t *testing.T) {
	t.Parallel()
	optional := NewOptional[testStruct](testStruct{value: 1})
	require.Equal(t, testStruct{value: 1}, optional.Value)
	require.True(t, optional.Present())
}

func TestStructPointerZero(t *testing.T) {
	t.Parallel()
	optional := NewOptional[*testStructPointer](nil)
	require.Nil(t, optional.Value)
	require.False(t, optional.Present())
}

func TestStructPointerPresent(t *testing.T) {
	t.Parallel()
	optional := NewOptional[*testStructPointer](&testStructPointer{})
	require.Equal(t, &testStructPointer{}, optional.Value)
	require.True(t, optional.Present())

	optional = NewOptional[*testStructPointer](&testStructPointer{value: 1})
	require.Equal(t, &testStructPointer{value: 1}, optional.Value)
	require.True(t, optional.Present())
}

func TestIfaceZero(t *testing.T) {
	t.Parallel()
	var testIfaceValue testIface
	optional := NewOptional[testIface](testIfaceValue)
	require.Equal(t, testIfaceValue, optional.Value)
	require.False(t, optional.Present())
}

func TestIfacePresent(t *testing.T) {
	t.Parallel()
	var testIfaceValue testIface = testStructFunc{}
	optional := NewOptional[testIface](testStructFunc{})
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())
	optional = NewOptional[testIface](testIfaceValue)
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())

	testIfaceValue = testStructFunc{value: 0}
	optional = NewOptional[testIface](testStructFunc{value: 0})
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())
	optional = NewOptional[testIface](testIfaceValue)
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())

	testIfaceValue = testStructFunc{value: 1}
	optional = NewOptional[testIface](testStructFunc{value: 1})
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())
	optional = NewOptional[testIface](testIfaceValue)
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())
}

func TestIfacePointerZero(t *testing.T) {
	t.Parallel()
	optional := NewOptional[testIface](nil)
	require.Nil(t, optional.Value)
	require.False(t, optional.Present())

	testIfaceValue := getNilTestIfaceValue()
	optional = NewOptional[testIface](testIfaceValue)
	require.Nil(t, optional.Value)
	require.False(t, optional.Present())

	testIfaceValue = getNilTestStructPointerValue()
	optional = NewOptional[testIface](testIfaceValue)
	require.Nil(t, optional.Value)
	// TODO: Why is this true?
	//require.False(t, optional.Present())
}

func TestIfacePointerPresent(t *testing.T) {
	t.Parallel()
	var testIfaceValue testIface = &testStructPointer{}
	optional := NewOptional[testIface](&testStructPointer{})
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())
	optional = NewOptional[testIface](testIfaceValue)
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())

	testIfaceValue = &testStructPointer{value: 1}
	optional = NewOptional[testIface](&testStructPointer{value: 1})
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())
	optional = NewOptional[testIface](testIfaceValue)
	require.Equal(t, testIfaceValue, optional.Value)
	require.True(t, optional.Present())
}

type testStruct struct {
	value int
}

func (t testStruct) Value() int {
	return t.value
}

type testStructFunc struct {
	value    int
	someFunc func() (testStructFunc, error)
}

func (t testStructFunc) Value() int {
	return t.value
}

type testStructPointer struct {
	value    int
	someFunc func() (*testStructPointer, error)
}

func (t *testStructPointer) Value() int {
	return t.value
}

type testIface interface {
	Value() int
}

func getNilTestIfaceValue() testIface {
	return nil
}

func getNilTestStructPointerValue() *testStructPointer {
	return nil
}
