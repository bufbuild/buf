package apphttp

import (
	"fmt"
	"net/http"
)

func setBasicAuth(
	request *http.Request,
	username string,
	password string,
	usernameKey string,
	passwordKey string,
) (bool, error) {
	if username != "" && password != "" {
		request.SetBasicAuth(username, password)
		return true, nil
	}
	if username == "" && password == "" {
		return false, nil
	}
	if password == "" {
		return false, fmt.Errorf("%s set but %s not set", usernameKey, passwordKey)
	}
	return false, fmt.Errorf("%s set but %s not set", passwordKey, usernameKey)
}
