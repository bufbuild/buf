// +build darwin linux

package clinetrc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMachineByName(t *testing.T) {
	testGetMachineByNameSuccess(
		t,
		"foo.com",
		"testdata/unix/home1",
		"foo.com",
		"bar",
		"baz",
		"bat",
	)
	testGetMachineByNameNil(
		t,
		"api.foo.com",
		"testdata/unix/home1",
	)
	testGetMachineByNameNil(
		t,
		"bar.com",
		"testdata/unix/home1",
	)
	testGetMachineByNameSuccess(
		t,
		"foo.com",
		"testdata/unix/home2",
		"",
		"bar",
		"baz",
		"",
	)
	testGetMachineByNameSuccess(
		t,
		"api.foo.com",
		"testdata/unix/home2",
		"",
		"bar",
		"baz",
		"",
	)
	testGetMachineByNameSuccess(
		t,
		"bar.com",
		"testdata/unix/home2",
		"",
		"bar",
		"baz",
		"",
	)
	testGetMachineByNameNil(
		t,
		"foo.com",
		"testdata/unix/home3",
	)
	testGetMachineByNameNil(
		t,
		"api.foo.com",
		"testdata/unix/home3",
	)
	testGetMachineByNameNil(
		t,
		"bar.com",
		"testdata/unix/home1",
	)
}

func testGetMachineByNameSuccess(
	t *testing.T,
	name string,
	home string,
	expectedName string,
	expectedLogin string,
	expectedPassword string,
	expectedAccount string,
) {
	machine, err := GetMachineByName(name, testNewGetenv(home))
	require.NoError(t, err)
	require.NotNil(t, machine)
	assert.Equal(t, expectedName, machine.Name)
	assert.Equal(t, expectedLogin, machine.Login)
	assert.Equal(t, expectedPassword, machine.Password)
	assert.Equal(t, expectedAccount, machine.Account)
}

func testGetMachineByNameNil(
	t *testing.T,
	name string,
	home string,
) {
	machine, err := GetMachineByName(name, testNewGetenv(home))
	require.NoError(t, err)
	require.Nil(t, machine)
}

func testNewGetenv(home string) func(string) string {
	return func(key string) string {
		if key == "HOME" {
			return home
		}
		return ""
	}
}
