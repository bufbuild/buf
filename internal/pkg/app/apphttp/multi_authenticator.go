package apphttp

import (
	"net/http"

	"github.com/bufbuild/buf/internal/pkg/app"
)

type multiAuthenticator struct {
	authenticators []Authenticator
}

func newMultiAuthenticator(authenticators ...Authenticator) *multiAuthenticator {
	return &multiAuthenticator{
		authenticators: authenticators,
	}
}

func (a *multiAuthenticator) SetAuth(envContainer app.EnvContainer, request *http.Request) (bool, error) {
	switch len(a.authenticators) {
	case 0:
		return false, nil
	case 1:
		return a.authenticators[0].SetAuth(envContainer, request)
	default:
		for _, authenticator := range a.authenticators {
			ok, err := authenticator.SetAuth(envContainer, request)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
}
