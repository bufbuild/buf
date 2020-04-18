package appcmd

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		BindFlags: func(flagSet *pflag.FlagSet) {
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
					data, err := ioutil.ReadAll(container.Stdin())
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
	require.NoError(t, Run(context.Background(), container, rootCommand, "0.1.0"))
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
	require.Equal(t, app.NewError(5, "bar"), Run(context.Background(), container, rootCommand, "0.1.0"))
}
