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

package bufsync

import (
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

type syncableModule struct {
	dir            string
	remoteIdentity bufmoduleref.ModuleIdentity
	formatted      string
}

func newSyncableModule(
	dir string,
	remoteIdentity bufmoduleref.ModuleIdentity,
) (Module, error) {
	normalized, err := normalpath.NormalizeAndValidate(dir)
	if err != nil {
		return nil, err
	}
	return &syncableModule{
		dir:            normalized,
		remoteIdentity: remoteIdentity,
		formatted:      fmt.Sprintf("%s:%s", dir, remoteIdentity.IdentityString()),
	}, nil
}

func (s *syncableModule) Dir() string {
	return s.dir
}

func (s *syncableModule) RemoteIdentity() bufmoduleref.ModuleIdentity {
	return s.remoteIdentity
}

func (s *syncableModule) String() string {
	return s.formatted
}
