// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufremotepluginref

import (
	"fmt"
	"math"
	"strconv"

	"golang.org/x/mod/semver"
)

type pluginReference struct {
	identity PluginIdentity
	version  string
	revision int
}

func (p *pluginReference) Remote() string {
	return p.identity.Remote()
}

func (p *pluginReference) Owner() string {
	return p.identity.Owner()
}

func (p *pluginReference) Plugin() string {
	return p.identity.Plugin()
}

func (p *pluginReference) IdentityString() string {
	return p.identity.IdentityString()
}

func (p *pluginReference) isPluginIdentity() {}

func (p *pluginReference) ReferenceString() string {
	return p.identity.IdentityString() + ":" + p.version + ":" + strconv.Itoa(p.revision)
}

func (p *pluginReference) Version() string {
	return p.version
}

func (p *pluginReference) Revision() int {
	return p.revision
}

func (p *pluginReference) isPluginReference() {}

func newPluginReference(identity PluginIdentity, version string, revision int) (*pluginReference, error) {
	if err := ValidatePluginIdentity(identity); err != nil {
		return nil, err
	}
	if err := ValidatePluginVersion(version); err != nil {
		return nil, err
	}
	if err := validatePluginRevision(revision); err != nil {
		return nil, err
	}
	return &pluginReference{
		identity: identity,
		version:  version,
		revision: revision,
	}, nil
}

func ValidatePluginVersion(version string) error {
	if !semver.IsValid(version) {
		return fmt.Errorf("plugin version %q is not a valid semantic version", version)
	}
	return nil
}

func validatePluginRevision(revision int) error {
	if revision < 0 || revision > math.MaxInt32 {
		return fmt.Errorf("revision %d is out of accepted range %d-%d", revision, 0, math.MaxInt32)
	}
	return nil
}
