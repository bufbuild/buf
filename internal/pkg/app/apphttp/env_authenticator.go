package apphttp

import (
	"net/http"

	"github.com/bufbuild/buf/internal/pkg/app"
)

type envAuthenticator struct {
	usernameKey string
	passwordKey string
}

func newEnvAuthenticator(
	usernameKey string,
	passwordKey string,
) *envAuthenticator {
	return &envAuthenticator{
		usernameKey: usernameKey,
		passwordKey: passwordKey,
	}
}

func (a *envAuthenticator) SetAuth(envContainer app.EnvContainer, request *http.Request) (bool, error) {
	return setBasicAuth(
		request,
		envContainer.Env(a.usernameKey),
		envContainer.Env(a.passwordKey),
		a.usernameKey,
		a.passwordKey,
	)
}
