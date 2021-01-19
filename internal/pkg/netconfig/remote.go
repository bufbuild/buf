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

type remote struct {
	address string
	login   *login
}

func newRemote(address string, token string) (*remote, error) {
	login, err := newLogin(token)
	if err != nil {
		return nil, err
	}
	remote := &remote{
		address: address,
		login:   login,
	}
	if err := validateRemote(remote); err != nil {
		return nil, err
	}
	return remote, nil
}

func (r *remote) Address() string {
	return r.address
}

func (r *remote) Login() (Login, bool) {
	if r.login == nil {
		return nil, false
	}
	return r.login, true
}

func (r *remote) toExternalRemote() ExternalRemote {
	externalRemote := ExternalRemote{
		Address: r.address,
	}
	if r.login != nil {
		externalRemote.Login = r.login.toExternalLogin()
	}
	return externalRemote
}

// validateRemote determines if all the required fields are specified.
func validateRemote(remote *remote) error {
	if remote == nil {
		return errors.New("remote cannot be nil")
	}
	if remote.address == "" {
		return errors.New("remote is missing an address")
	}
	return nil
}
