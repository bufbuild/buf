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

package appcmd

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bufbuild/buf/private/pkg/app"
)

func TestBasic(t *testing.T) {
	var foo string
	var bar int

	var actualArgs []string
	var actualFoo string
	var actualBar int
	var actualStdin string
	var actualEnvValue string

	rootCommand := &Command{
		Use: "test",
		BindPersistentFlags: func(flagSet *pflag.FlagSet) {
			flagSet.StringVar(&foo, "foo", "", "Foo.")
		},
		SubCommands: []*Command{
			{
				Use: "sub",
				BindFlags: func(flagSet *pflag.FlagSet) {
					flagSet.IntVar(&bar, "bar", 1, "Bar.")
				},
				Run: func(ctx context.Context, container app.Container) error {
					actualArgs = app.Args(container)
					actualFoo = foo
					actualBar = bar
					data, err := io.ReadAll(container.Stdin())
					if err != nil {
						return err
					}
					actualStdin = string(data)
					actualEnvValue = container.Env("KEY")
					return nil
				},
			},
		},
	}
	container := app.NewContainer(
		map[string]string{
			"KEY": "VALUE",
		},
		strings.NewReader("world"),
		nil,
		nil,
		"test",
		"sub",
		"one",
		"two",
		"--foo",
		"hello",
	)
	require.NoError(t, Run(context.Background(), container, rootCommand))
	assert.Equal(t, []string{"one", "two"}, actualArgs)
	assert.Equal(t, "hello", actualFoo)
	assert.Equal(t, 1, actualBar)
	assert.Equal(t, "world", actualStdin)
	assert.Equal(t, "VALUE", actualEnvValue)
}

func TestError(t *testing.T) {
	rootCommand := &Command{
		Use: "test",
		SubCommands: []*Command{
			{
				Use: "sub",
				Run: func(ctx context.Context, container app.Container) error {
					return app.NewError(5, "bar")
				},
			},
		},
	}
	container := app.NewContainer(
		nil,
		nil,
		nil,
		nil,
		"test",
		"sub",
	)
	require.Equal(t, app.NewError(5, "bar"), Run(context.Background(), container, rootCommand))
}

func TestVersionToStdout(t *testing.T) {
	version := "0.0.1-dev"
	rootCommand := &Command{
		Use:     "test",
		Version: version,
		SubCommands: []*Command{
			{
				Use: "foo",
				Run: func(context.Context, app.Container) error {
					return nil
				},
			},
		},
	}
	buffer := bytes.NewBuffer(nil)
	container := app.NewContainer(
		nil,
		nil,
		buffer,
		nil,
		"test",
		"--version",
	)
	require.NoError(t, Run(context.Background(), container, rootCommand))
	require.Equal(t, version+"\n", buffer.String())

	rootCommand = &Command{
		Use:     "test",
		Version: version,
		Run: func(context.Context, app.Container) error {
			return nil
		},
	}
	buffer = bytes.NewBuffer(nil)
	container = app.NewContainer(
		nil,
		nil,
		buffer,
		nil,
		"test",
		"--version",
	)
	require.NoError(t, Run(context.Background(), container, rootCommand))
	require.Equal(t, version+"\n", buffer.String())
}

func TestHelpToStdout(t *testing.T) {
	rootCommand := &Command{
		Use: "test",
		// need a sub-command for "help" to work
		// otherwise can do -h
		SubCommands: []*Command{
			{
				Use: "foo",
				Run: func(context.Context, app.Container) error {
					return nil
				},
			},
		},
	}
	buffer := bytes.NewBuffer(nil)
	container := app.NewContainer(
		nil,
		nil,
		buffer,
		nil,
		"test",
		"help",
	)
	require.NoError(t, Run(context.Background(), container, rootCommand))
	require.NotEmpty(t, buffer.String())

	rootCommand = &Command{
		Use: "test",
		Run: func(context.Context, app.Container) error {
			return nil
		},
	}
	buffer = bytes.NewBuffer(nil)
	container = app.NewContainer(
		nil,
		nil,
		buffer,
		nil,
		"test",
		"-h",
	)
	require.NoError(t, Run(context.Background(), container, rootCommand))
	require.NotEmpty(t, buffer.String())
}

func TestIncorrectFlagEmptyStdout(t *testing.T) {
	rootCommand := &Command{
		Use: "test",
		Run: func(context.Context, app.Container) error {
			return nil
		},
	}
	stderr := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	container := app.NewContainer(
		nil,
		nil,
		stdout,
		stderr,
		"test",
		"--foo",
		"1",
	)
	require.Error(t, Run(context.Background(), container, rootCommand))
	require.Empty(t, stdout.String())
	require.NotEmpty(t, stderr.String())
}
