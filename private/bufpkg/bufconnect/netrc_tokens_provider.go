// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufconnect

import (
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/netrc"
)

// netrcTokensProvider is used to provide remote tokens from .netrc.
type netrcTokensProvider struct {
	container         app.EnvContainer
	getMachineForName func(app.EnvContainer, string) (netrc.Machine, error)
}

var _ TokenProvider = (*netrcTokensProvider)(nil)

// NewNetrcTokensProvider returns a TokenProvider for a .netrc in a container.
func NewNetrcTokensProvider(container app.EnvContainer, getMachineForName func(app.EnvContainer, string) (netrc.Machine, error)) TokenProvider {
	return &netrcTokensProvider{container: container, getMachineForName: getMachineForName}
}

func (nt *netrcTokensProvider) RemoteToken(address string) string {
	machine, err := nt.getMachineForName(nt.container, address)
	if err != nil {
		return ""
	}
	if machine != nil {
		return machine.Password()
	}
	return ""
}

func (nt *netrcTokensProvider) IsFromEnvVar() bool {
	return false
}
