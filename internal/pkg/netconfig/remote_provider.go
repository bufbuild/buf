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
	"fmt"
	"sort"
)

type remoteProvider struct {
	addressToRemote map[string]Remote
}

func newRemoteProvider(externalRemotes []ExternalRemote) (*remoteProvider, error) {
	addressToRemote := make(map[string]Remote, len(externalRemotes))
	for _, externalRemote := range externalRemotes {
		remote, err := newRemote(
			externalRemote.Address,
			externalRemote.Login.Token,
		)
		if err != nil {
			return nil, err
		}
		if _, ok := addressToRemote[remote.Address()]; ok {
			return nil, fmt.Errorf("remote address %s is specified more than once", remote.Address())
		}
		addressToRemote[remote.Address()] = remote
	}
	return newRemoteProviderInternal(addressToRemote), nil
}

func newRemoteProviderInternal(addressToRemote map[string]Remote) *remoteProvider {
	return &remoteProvider{
		addressToRemote: addressToRemote,
	}
}

func (r *remoteProvider) GetRemote(address string) (Remote, bool) {
	remote, ok := r.addressToRemote[address]
	return remote, ok
}

func (r *remoteProvider) WithUpdatedRemote(address string, updatedToken string) (RemoteProvider, error) {
	updatedRemote, err := newRemote(
		address,
		updatedToken,
	)
	if err != nil {
		return nil, err
	}
	newAddressToRemote := map[string]Remote{
		address: updatedRemote,
	}
	for curAddress, curRemote := range r.addressToRemote {
		if curAddress != address {
			newAddressToRemote[curAddress] = curRemote
		}
	}
	return newRemoteProviderInternal(newAddressToRemote), nil
}

func (r *remoteProvider) WithoutRemote(address string) (RemoteProvider, bool) {
	found := false
	newAddressToRemote := make(map[string]Remote, len(r.addressToRemote))
	for curAddress, curRemote := range r.addressToRemote {
		if curAddress == address {
			found = true
		} else {
			newAddressToRemote[curAddress] = curRemote
		}
	}
	if !found {
		return r, false
	}
	return newRemoteProviderInternal(newAddressToRemote), true
}

func (r *remoteProvider) ToExternalRemotes() []ExternalRemote {
	externalRemotes := make([]ExternalRemote, 0, len(r.addressToRemote))
	for _, remote := range r.addressToRemote {
		externalRemotes = append(externalRemotes, remote.toExternalRemote())
	}
	sort.Slice(
		externalRemotes,
		func(i int, j int) bool {
			return externalRemotes[i].Address < externalRemotes[j].Address
		},
	)
	return externalRemotes
}
