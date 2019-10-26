package internal

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/bufbuild/buf/internal/pkg/cli"
	"github.com/bufbuild/buf/internal/pkg/errs"
	ioutilext "github.com/bufbuild/buf/internal/pkg/ioutil"
)

// SetRunEnvDefaults sets the defaults for the RunEnv.
func SetRunEnvDefaults(runEnv *cli.RunEnv) {
	if runEnv.Args == nil {
		runEnv.Args = []string{}
	}
	if runEnv.Stdin == nil {
		runEnv.Stdin = ioutilext.DiscardReader
	}
	if runEnv.Stdout == nil {
		runEnv.Stdout = ioutil.Discard
	}
	if runEnv.Stderr == nil {
		runEnv.Stderr = ioutil.Discard
	}
	if runEnv.Environ == nil {
		runEnv.Environ = []string{}
	}
}

// NewOSRunEnv returns a new OS RunEnv.
func NewOSRunEnv() *cli.RunEnv {
	return &cli.RunEnv{
		Args:    os.Args[1:],
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Environ: os.Environ(),
	}
}

// NewExecEnv returns a new ExecEnv.
//
// Args should not include the application name.
func NewExecEnv(
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	environ []string,
	start time.Time,
) (*cli.ExecEnv, error) {
	env, err := environToEnv(environ)
	if err != nil {
		return nil, err
	}
	return &cli.ExecEnv{
		Args:   args,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Env:    env,
		Start:  start,
	}, nil
}

func environToEnv(environ []string) (map[string]string, error) {
	env := make(map[string]string, len(environ))
	for _, elem := range environ {
		if !strings.ContainsRune(elem, '=') {
			return nil, errs.NewInternalf("environment variable does not contain =")
		}
		split := strings.SplitN(elem, "=", 2)
		switch len(split) {
		case 1:
			env[split[0]] = ""
		case 2:
			env[split[0]] = split[1]
		default:
			return nil, errs.NewInternalf("unknown environment split")
		}
	}
	return env, nil
}
