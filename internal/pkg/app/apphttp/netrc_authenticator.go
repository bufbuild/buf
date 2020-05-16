package apphttp

import (
	"errors"
	"net/http"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appnetrc"
)

type netrcAuthenticator struct{}

func newNetrcAuthenticator() *netrcAuthenticator {
	return &netrcAuthenticator{}
}

func (a *netrcAuthenticator) SetAuth(envContainer app.EnvContainer, request *http.Request) (bool, error) {
	if request.URL == nil {
		return false, errors.New("malformed request: no url")
	}
	if request.URL.Host == "" {
		return false, errors.New("malformed request: no url host")
	}
	machine, err := appnetrc.GetMachineForName(envContainer, request.URL.Host)
	if err != nil {
		return false, err
	}
	if machine == nil {
		return false, nil
	}
	return setBasicAuth(
		request,
		machine.Login(),
		machine.Password(),
		"netrc login for host",
		"netrc password for host",
	)
}
