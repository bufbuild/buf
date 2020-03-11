package clienv

import (
	"io"
	"io/ioutil"

	internalioutil "github.com/bufbuild/buf/internal/pkg/cli/internal/ioutil"
)

type env struct {
	args      []string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	variables map[string]string
}

func newEnv(
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	variables map[string]string,
) *env {
	env := &env{
		args:      args,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		variables: variables,
	}
	if env.args == nil {
		env.args = []string{}
	}
	if env.stdin == nil {
		env.stdin = internalioutil.DiscardReader
	}
	if env.stdout == nil {
		env.stdout = ioutil.Discard
	}
	if env.stderr == nil {
		env.stderr = ioutil.Discard
	}
	if env.variables == nil {
		env.variables = make(map[string]string)
	}
	return env
}

func (e *env) Args() []string {
	return e.args
}

func (e *env) Stdin() io.Reader {
	return e.stdin
}

func (e *env) Stdout() io.Writer {
	return e.stdout
}

func (e *env) Stderr() io.Writer {
	return e.stderr
}

func (e *env) Getenv(key string) string {
	return e.variables[key]
}

func (e *env) WithArgs(args []string) Env {
	return newEnv(
		args,
		e.stdin,
		e.stdout,
		e.stderr,
		e.variables,
	)
}
