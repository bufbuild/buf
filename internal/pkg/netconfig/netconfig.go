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

// Package netconfig is roughly analogous to netrc but with json or yaml configuration
// and more validation.
package netconfig

// ExternalRemote represents a remote configuration in json or yaml.
type ExternalRemote struct {
	Address string        `json:"address,omitempty" yaml:"address,omitempty"`
	Login   ExternalLogin `json:"login,omitempty" yaml:"login,omitempty"`
}

// ExternalLogin represents a credentials configuration in json or yaml.
type ExternalLogin struct {
	Token string `json:"token,omitempty" yaml:"token,omitempty"`
}

// Remote represents the configuration required for a given address.
type Remote interface {
	Address() string
	Login() (Login, bool)

	toExternalRemote() ExternalRemote
}

// Login represents user credentials.
type Login interface {
	Token() string

	toExternalLogin() ExternalLogin
}

// RemoteProvider provides Remotes.
type RemoteProvider interface {
	// GetRemote gets the Remote for the address.
	//
	// Returns false if no such remote exists.
	GetRemote(address string) (Remote, bool)
	// WithUpdatedRemote returns a new RemoteProvider with the Remote updated at the given address.
	//
	// If this Remote already existed, this overwrites the existing remote.
	// If this Remote did not exist, this adds a new Remote.
	WithUpdatedRemote(address string, updatedToken string) (RemoteProvider, error)
	// WithoutRemote returns a new RemoteProvider with the Remove deleted.
	//
	// Returns false if the remote did not exist.
	WithoutRemote(address string) (RemoteProvider, bool)
	// ToExternalRemotes converts the RemoteProvider into sorted ExternalRemotes.
	//
	// Sorted by address.
	ToExternalRemotes() []ExternalRemote
}

// NewRemoteProvider returns a new RemoteProvider for the ExternalRemotes.
func NewRemoteProvider(externalRemotes []ExternalRemote) (RemoteProvider, error) {
	return newRemoteProvider(externalRemotes)
}
