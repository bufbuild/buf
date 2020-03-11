package clicobra

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/cli/clienv"
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
				Run: func(env clienv.Env) error {
					actualArgs = env.Args()
					actualFoo = foo
					actualBar = bar
					data, err := ioutil.ReadAll(env.Stdin())
					if err != nil {
						return err
					}
					actualStdin = string(data)
					actualEnvValue = env.Getenv("KEY")
					return nil
				},
			},
		},
	}
	env := clienv.NewEnv(
		[]string{
			"sub",
			"one",
			"two",
			"--foo",
			"hello",
		},
		strings.NewReader("world"),
		nil,
		nil,
		map[string]string{
			"KEY": "VALUE",
		},
	)
	require.Equal(t, Run(rootCommand, "0.1.0", env), 0)
	assert.Equal(t, []string{"one", "two"}, actualArgs)
	assert.Equal(t, "hello", actualFoo)
	assert.Equal(t, 1, actualBar)
	assert.Equal(t, "world", actualStdin)
	assert.Equal(t, "VALUE", actualEnvValue)
}
