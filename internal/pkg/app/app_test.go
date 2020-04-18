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

	envContainer, err := newEnvContainerForEnviron(
		[]string{
			"foo1=bar1",
			"foo2=bar2",
			"foo3=",
		},
	)
	require.NoError(t, err)
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
