// Copyright 2020 Buf Technologies, Inc.
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

// +build darwin linux

package netrc

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMachineForName(t *testing.T) {
	testGetMachineForNameSuccess(
		t,
		"foo.com",
		"testdata/unix/home1",
		"foo.com",
		"bar",
		"baz",
		"bat",
	)
	testGetMachineForNameNil(
		t,
		"api.foo.com",
		"testdata/unix/home1",
	)
	testGetMachineForNameNil(
		t,
		"bar.com",
		"testdata/unix/home1",
	)
	testGetMachineForNameSuccess(
		t,
		"foo.com",
		"testdata/unix/home2",
		"",
		"bar",
		"baz",
		"",
	)
	testGetMachineForNameSuccess(
		t,
		"api.foo.com",
		"testdata/unix/home2",
		"",
		"bar",
		"baz",
		"",
	)
	testGetMachineForNameSuccess(
		t,
		"bar.com",
		"testdata/unix/home2",
		"",
		"bar",
		"baz",
		"",
	)
	testGetMachineForNameNil(
		t,
		"foo.com",
		"testdata/unix/home3",
	)
	testGetMachineForNameNil(
		t,
		"api.foo.com",
		"testdata/unix/home3",
	)
	testGetMachineForNameNil(
		t,
		"bar.com",
		"testdata/unix/home1",
	)
}

func TestPutMachine(t *testing.T) {
	testPutMachineSuccess(
		t,
		"foo.com",
		"test@foo.com",
		"password",
		false,
	)
	testPutMachineSuccess(
		t,
		"foo.com",
		"test@foo.com",
		"password",
		true,
	)
	testPutMachineError(
		t,
		"foo.com",
		"test@foo.com",
		"password",
	)
}

func testPutMachineSuccess(
	t *testing.T,
	name string,
	login string,
	password string,
	createNetrcBeforePut bool,
) {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), netrcFilename)
	envContainer := app.NewEnvContainer(map[string]string{"NETRC": filePath})
	machine, err := GetMachineForName(envContainer, name)
	require.NoError(t, err)
	require.Nil(t, machine)

	if createNetrcBeforePut {
		_, err = os.Create(filePath)
		require.NoError(t, err)
	}

	expectedMachine := NewMachine(name, login, password, "")
	err = PutMachine(envContainer, expectedMachine)
	require.NoError(t, err)

	actualMachine, err := GetMachineForName(envContainer, name)
	require.NoError(t, err)
	assert.Equal(t, expectedMachine, actualMachine)
}

func testPutMachineError(
	t *testing.T,
	name string,
	login string,
	password string,
) {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), netrcFilename)
	envContainer := app.NewEnvContainer(map[string]string{"NETRC": filePath})
	_, err := os.Create(filePath)
	require.NoError(t, err)
	err = ioutil.WriteFile(filePath, []byte("invalid netrc"), 0644)
	require.NoError(t, err)
	err = PutMachine(envContainer, NewMachine(name, login, password, ""))
	require.Error(t, err)
}

func testGetMachineForNameSuccess(
	t *testing.T,
	name string,
	homeDirPath string,
	expectedName string,
	expectedLogin string,
	expectedPassword string,
	expectedAccount string,
) {
	t.Helper()
	machine, err := GetMachineForName(app.NewEnvContainer(map[string]string{"HOME": homeDirPath}), name)
	require.NoError(t, err)
	require.NotNil(t, machine)
	assert.Equal(t, expectedName, machine.Name())
	assert.Equal(t, expectedLogin, machine.Login())
	assert.Equal(t, expectedPassword, machine.Password())
	assert.Equal(t, expectedAccount, machine.Account())
}

func testGetMachineForNameNil(
	t *testing.T,
	name string,
	homeDirPath string,
) {
	t.Helper()
	machine, err := GetMachineForName(app.NewEnvContainer(map[string]string{"HOME": homeDirPath}), name)
	require.NoError(t, err)
	require.Nil(t, machine)
}
