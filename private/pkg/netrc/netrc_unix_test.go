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

// Matching the unix-like build tags in the Golang source i.e. https://github.com/golang/go/blob/912f0750472dd4f674b69ca1616bfaf377af1805/src/os/file_unix.go#L6

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package netrc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMachineForName(t *testing.T) {
	t.Parallel()
	testGetMachineForNameSuccess(
		t,
		"foo.com",
		"testdata/unix/home1",
		"foo.com",
		"bar",
		"baz",
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
	)
	testGetMachineForNameSuccess(
		t,
		"api.foo.com",
		"testdata/unix/home2",
		"",
		"bar",
		"baz",
	)
	testGetMachineForNameSuccess(
		t,
		"bar.com",
		"testdata/unix/home2",
		"",
		"bar",
		"baz",
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

func TestPutMachines(t *testing.T) {
	t.Parallel()
	testPutMachinesSuccess(
		t,
		false,
		NewMachine(
			"foo.com",
			"test@foo.com",
			"password",
		),
	)
	testPutMachinesSuccess(
		t,
		true,
		NewMachine(
			"foo.com",
			"test@foo.com",
			"password",
		),
	)
	testPutMachinesSuccess(
		t,
		false,
		NewMachine(
			"bar.com",
			"test@bar.com",
			"password",
		),
		NewMachine(
			"baz.com",
			"test@baz.com",
			"password",
		),
	)
}

// https://github.com/bufbuild/buf/issues/611
func TestPutLotsOfBigMachinesSingleLineFiles(t *testing.T) {
	t.Parallel()
	size := 10
	password := strings.Repeat("abcdefghijklmnopqrstuvwxyz", size)
	machines := make([]Machine, 0, size)
	buffer := bytes.NewBuffer(nil)
	for i := 0; i < size; i++ {
		// Write the file manually in single-line format as this is where the failure happens.
		_, _ = buffer.WriteString(fmt.Sprintf("machine foo%d login bar%d password %s\n", i, i, password))
		machines = append(
			machines,
			NewMachine(
				fmt.Sprintf("foo%d", i),
				fmt.Sprintf("bar%d", i),
				password,
			),
		)
	}
	filePath := filepath.Join(t.TempDir(), netrcFilename)
	err := os.WriteFile(filePath, buffer.Bytes(), 0600)
	require.NoError(t, err)

	envContainer := app.NewEnvContainer(map[string]string{"NETRC": filePath})
	for _, machine := range machines {
		// Make sure the existing file can be parsed.
		actualMachine, err := GetMachineForName(envContainer, machine.Name())
		require.NoError(t, err)
		require.Equal(t, machine, actualMachine)
	}

	// Now, modify the file with an extra machine. This is when the file got corrupted.
	extraMachine := NewMachine(
		"baz.com",
		"test@baz.com",
		"password",
	)
	err = PutMachines(envContainer, extraMachine)
	require.NoError(t, err)
	machines = append(machines, extraMachine)
	for _, machine := range machines {
		// Verify all the machines work. This failed previously.
		actualMachine, err := GetMachineForName(envContainer, machine.Name())
		require.NoError(t, err)
		require.Equal(t, machine, actualMachine)
	}

	// Modify some of the existing machines.
	machines = make([]Machine, 0, size)
	for i := 0; i < size; i++ {
		modifiedPassword := password
		if i%2 == 0 {
			modifiedPassword = modifiedPassword + "Z"
		}
		machine := NewMachine(
			fmt.Sprintf("foo%d", i),
			fmt.Sprintf("bar%d", i),
			modifiedPassword,
		)
		machines = append(machines, machine)
		if i%2 == 0 {
			err = PutMachines(envContainer, machine)
			require.NoError(t, err)
		}
	}
	machines = append(machines, extraMachine)
	for _, machine := range machines {
		actualMachine, err := GetMachineForName(envContainer, machine.Name())
		require.NoError(t, err)
		require.Equal(t, machine, actualMachine)
	}
}

func TestDeleteMachineForName(t *testing.T) {
	t.Parallel()
	filePath := filepath.Join(t.TempDir(), netrcFilename)
	envContainer := app.NewEnvContainer(map[string]string{"NETRC": filePath})
	err := PutMachines(
		envContainer,
		NewMachine(
			"bar.com",
			"test@bar.com",
			"password",
		),
		NewMachine(
			"baz.com",
			"test@baz.com",
			"password",
		),
	)
	require.NoError(t, err)
	exists, err := DeleteMachineForName(envContainer, "bar.com")
	require.NoError(t, err)
	require.True(t, exists)
	machine, err := GetMachineForName(envContainer, "bar.com")
	require.NoError(t, err)
	assert.Nil(t, machine)
	machine, err = GetMachineForName(envContainer, "baz.com")
	require.NoError(t, err)
	assert.NotNil(t, machine)
	exists, err = DeleteMachineForName(envContainer, "bar.com")
	require.NoError(t, err)
	require.False(t, exists)
}

func testPutMachinesSuccess(
	t *testing.T,
	createNetrcBeforePut bool,
	machines ...Machine,
) {
	filePath := filepath.Join(t.TempDir(), netrcFilename)
	envContainer := app.NewEnvContainer(map[string]string{"NETRC": filePath})
	for _, machine := range machines {
		machine, err := GetMachineForName(envContainer, machine.Name())
		require.NoError(t, err)
		require.Nil(t, machine)
	}

	if createNetrcBeforePut {
		_, err := os.Create(filePath)
		require.NoError(t, err)
	}

	err := PutMachines(envContainer, machines...)
	require.NoError(t, err)

	for _, machine := range machines {
		actualMachine, err := GetMachineForName(envContainer, machine.Name())
		require.NoError(t, err)
		assert.Equal(t, machine, actualMachine)
	}
}

func testGetMachineForNameSuccess(
	t *testing.T,
	name string,
	homeDirPath string,
	expectedName string,
	expectedLogin string,
	expectedPassword string,
) {
	machine, err := GetMachineForName(app.NewEnvContainer(map[string]string{"HOME": homeDirPath}), name)
	require.NoError(t, err)
	require.NotNil(t, machine)
	assert.Equal(t, expectedName, machine.Name())
	assert.Equal(t, expectedLogin, machine.Login())
	assert.Equal(t, expectedPassword, machine.Password())
}

func testGetMachineForNameNil(
	t *testing.T,
	name string,
	homeDirPath string,
) {
	machine, err := GetMachineForName(app.NewEnvContainer(map[string]string{"HOME": homeDirPath}), name)
	require.NoError(t, err)
	require.Nil(t, machine)
}
