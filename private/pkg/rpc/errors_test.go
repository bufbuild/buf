// Copyright 2020-2022 Buf Technologies, Inc.
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

package rpc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetErrorMessage(t *testing.T) {
	require.Equal(t, "test", GetErrorMessage(fmt.Errorf("some error: %w", NewInvalidArgumentError("test"))))
}

func TestErrorsIs(t *testing.T) {
	cancelError := NewCanceledError("cancel")
	assert.True(
		t,
		errors.Is(cancelError, cancelError),
	)
	assert.True(
		t,
		errors.Is(fmt.Errorf("wrap this: %w", cancelError), cancelError),
	)
	assert.False(
		t,
		errors.Is(NewCanceledError("cancel"), cancelError),
	)
	assert.False(
		t,
		errors.Is(NewCanceledError("cancel"), NewCanceledError("other message")),
	)
	assert.False(
		t,
		errors.Is(NewCanceledError("cancel"), NewAbortedError("cancel")),
	)
}

func TestIsError(t *testing.T) {
	assert.True(t, IsError(NewError(ErrorCodeOK, "something")))
	assert.False(t, IsError(errors.New("something")))
	assert.False(t, IsError(nil))
}
