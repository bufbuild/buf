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

package bufapp

import (
	"crypto/tls"
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/app/appname"
	"github.com/bufbuild/buf/internal/pkg/cert/certclient"
	"github.com/bufbuild/buf/internal/pkg/netconfig"
)

const currentVersion = "v1"

// ExternalConfig is an external config.
type ExternalConfig struct {
	// if editing ExternalConfig, make sure to update externalConfigIsEmpty at the bottom of this file!

	Version string                             `json:"version,omitempty" yaml:"version,omitempty"`
	TLS     certclient.ExternalClientTLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
	Remotes []netconfig.ExternalRemote         `json:"remotes,omitempty" yaml:"remotes,omitempty"`
}

// IsEmpty returns true if the externalConfig is empty.
func (e ExternalConfig) IsEmpty() bool {
	// you can't just do externalConfig == ExternalConfig{} as Golang does not allow
	// this if you have a slice field, i.e. Remotes
	return e.Version == "" &&
		e.TLS.IsEmpty() &&
		len(e.Remotes) == 0
}

// Config is a config.
type Config struct {
	TLS            *tls.Config
	RemoteProvider netconfig.RemoteProvider
}

// NewConfig returns a new Config for the ExternalConfig.
func NewConfig(
	container appname.Container,
	externalConfig ExternalConfig,
) (*Config, error) {
	if externalConfig.Version != currentVersion && !externalConfig.IsEmpty() {
		return nil, fmt.Errorf("buf configuration at %q must declare 'version: %s'", container.ConfigDirPath(), currentVersion)
	}
	tlsConfig, err := certclient.NewClientTLSConfig(container, externalConfig.TLS)
	if err != nil {
		return nil, err
	}
	remoteProvider, err := netconfig.NewRemoteProvider(externalConfig.Remotes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remotes configuration at %q: %v", container.ConfigDirPath(), err)
	}
	return &Config{
		TLS:            tlsConfig,
		RemoteProvider: remoteProvider,
	}, nil
}
