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

package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvContainer(t *testing.T) {
	envContainer := NewEnvContainer(
		map[string]string{
			"foo1": "bar1",
			"foo2": "bar2",
			"foo3": "",
		},
	)
	assert.Equal(t, "bar1", envContainer.Env("foo1"))
	assert.Equal(t, "bar2", envContainer.Env("foo2"))
	assert.Equal(t, "", envContainer.Env("foo3"))
	assert.Equal(
		t,
		[]string{
			"foo1=bar1",
			"foo2=bar2",
		},
		Environ(envContainer),
	)
	assert.Equal(
		t,
		map[string]string{
			"foo1": "bar1",
			"foo2": "bar2",
		},
		EnvironMap(envContainer),
	)

	envContainer, err := newEnvContainerForEnviron(
		[]string{
			"foo1=bar1",
			"foo2=bar2",
			"foo3=bar3",
			"foo4=",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "bar1", envContainer.Env("foo1"))
	assert.Equal(t, "bar2", envContainer.Env("foo2"))
	assert.Equal(t, "bar3", envContainer.Env("foo3"))
	assert.Equal(t, "", envContainer.Env("foo4"))
	assert.Equal(
		t,
		[]string{
			"foo1=bar1",
			"foo2=bar2",
			"foo3=bar3",
		},
		Environ(envContainer),
	)
	assert.Equal(
		t,
		map[string]string{
			"foo1": "bar1",
			"foo2": "bar2",
			"foo3": "bar3",
		},
		EnvironMap(envContainer),
	)

	envContainer = NewEnvContainerWithOverrides(
		envContainer,
		map[string]string{
			"foo1": "",
			"foo2": "baz2",
		},
	)
	assert.Equal(t, "", envContainer.Env("foo1"))
	assert.Equal(t, "baz2", envContainer.Env("foo2"))
	assert.Equal(t, "bar3", envContainer.Env("foo3"))
	assert.Equal(t, "", envContainer.Env("foo4"))
	assert.Equal(
		t,
		[]string{
			"foo2=baz2",
			"foo3=bar3",
		},
		Environ(envContainer),
	)
	assert.Equal(
		t,
		map[string]string{
			"foo2": "baz2",
			"foo3": "bar3",
		},
		EnvironMap(envContainer),
	)

	_, err = newEnvContainerForEnviron(
		[]string{
			"foo1=bar1",
			"foo2=bar2",
			"foo3",
		},
	)
	require.Error(t, err)
}

func TestArgContainer(t *testing.T) {
	args := []string{"foo", "bar", "baz"}
	assert.Equal(t, args, Args(NewArgContainer(args...)))
}

func TestIsDev(t *testing.T) {
	assert.Equal(t, DevStdinFilePath != "", IsDevStdin(DevStdinFilePath))
	assert.Equal(t, DevStdoutFilePath != "", IsDevStdout(DevStdoutFilePath))
	assert.Equal(t, DevStderrFilePath != "", IsDevStderr(DevStderrFilePath))
	assert.Equal(t, DevNullFilePath != "", IsDevNull(DevNullFilePath))
	assert.False(t, IsDevStdin("foo"))
	assert.False(t, IsDevStdout("foo"))
	assert.False(t, IsDevStderr("foo"))
	assert.False(t, IsDevNull("foo"))
}

func TestGetEnvBoolValue(t *testing.T) {
	envContainer := NewEnvContainer(
		map[string]string{
			"foo1": "bar1",
			"foo2": "true",
			"foo3": "false",
		},
	)
	val, err := envContainer.GetEnvBoolValue("foo1")
	assert.Error(t, err)
	assert.Equal(t, false, val)
	val, err = envContainer.GetEnvBoolValue("foo2")
	assert.NoError(t, err)
	assert.Equal(t, true, val)
	val, err = envContainer.GetEnvBoolValue("foo3")
	assert.NoError(t, err)
	assert.Equal(t, false, val)
}
