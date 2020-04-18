// +build darwin linux

package appnetrc

import (
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

func testGetMachineForNameSuccess(
	t *testing.T,
	name string,
	homeDirPath string,
	expectedName string,
	expectedLogin string,
	expectedPassword string,
	expectedAccount string,
) {
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
	machine, err := GetMachineForName(app.NewEnvContainer(map[string]string{"HOME": homeDirPath}), name)
	require.NoError(t, err)
	require.Nil(t, machine)
}
