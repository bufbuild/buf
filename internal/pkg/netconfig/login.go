// Copyright 2020-2021 Buf Technologies, Inc.
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

package netconfig

import (
	"errors"
)

type login struct {
	token string
}

// returns nil if the token is empty.
func newLogin(token string) (*login, error) {
	if token == "" {
		return nil, nil
	}
	login := &login{
		token: token,
	}
	if err := validateLogin(login); err != nil {
		return nil, err
	}
	return login, nil
}

func (r *login) Token() string {
	return r.token
}

func (r *login) toExternalLogin() ExternalLogin {
	return ExternalLogin{
		Token: r.token,
	}
}

// validateLogin determines if all the required fields are specified.
func validateLogin(login *login) error {
	if login == nil {
		return errors.New("login cannot be nil")
	}
	if login.token == "" {
		return errors.New("login is missing a token")
	}
	return nil
}
