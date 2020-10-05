// Copyright 2020 Buf Technologies, Inc.
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

package bufmodule

import (
	"fmt"

	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
)

// resolvedModuleName implements the ResolvedModuleName interface.
type resolvedModuleName struct {
	*moduleName
}

func newResolvedModuleName(
	remote string,
	owner string,
	repository string,
	version string,
	digest string,
) (*resolvedModuleName, error) {
	return newResolvedModuleNameForProto(
		&modulev1.ModuleName{
			Remote:     remote,
			Owner:      owner,
			Repository: repository,
			Version:    version,
			Digest:     digest,
		},
	)
}

func newResolvedModuleNameForProto(
	protoModuleName *modulev1.ModuleName,
) (*resolvedModuleName, error) {
	moduleName, err := newModuleNameForProto(protoModuleName)
	if err != nil {
		return nil, err
	}
	if moduleName.digest == "" {
		return nil, NewNoDigestError(moduleName)
	}
	return &resolvedModuleName{
		moduleName: moduleName,
	}, nil
}

func (m *resolvedModuleName) String() string {
	base := moduleNameIdentity(m.moduleName)
	return fmt.Sprintf("%s:%s", base, m.digest)
}

func (m *resolvedModuleName) isResolvedModuleName() {}
