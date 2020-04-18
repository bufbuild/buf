package app

import (
	"errors"
	"strings"
)

type envContainer struct {
	variables map[string]string
}

func newEnvContainer(m map[string]string) *envContainer {
	variables := make(map[string]string)
	for key, value := range m {
		if value != "" {
			variables[key] = value
		}
	}
	return &envContainer{
		variables: variables,
	}
}

func newEnvContainerForEnviron(environ []string) (*envContainer, error) {
	variables := make(map[string]string, len(environ))
	for _, elem := range environ {
		if !strings.ContainsRune(elem, '=') {
			// Do not print out as we don't want to mistakenly leak a secure environment variable
			return nil, errors.New("environment variable does not contain =")
		}
		split := strings.SplitN(elem, "=", 2)
		if len(split) != 2 {
			// Do not print out as we don't want to mistakenly leak a secure environment variable
			return nil, errors.New("unknown environment split")
		}
		if split[1] != "" {
			variables[split[0]] = split[1]
		}
	}
	return &envContainer{
		variables: variables,
	}, nil
}

func (e *envContainer) Env(key string) string {
	return e.variables[key]
}

func (e *envContainer) ForEachEnv(f func(string, string)) {
	for key, value := range e.variables {
		f(key, value)
	}
}
