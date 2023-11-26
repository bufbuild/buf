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

package bufmoduletesting

import (
	"context"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
)

type testModuleReader struct {
	moduleIdentityStringToModule map[string]bufmodule.Module
}

func newTestModuleReader(moduleIdentityStringToModule map[string]bufmodule.Module) *testModuleReader {
	return &testModuleReader{
		moduleIdentityStringToModule: moduleIdentityStringToModule,
	}
}

func (r *testModuleReader) GetModule(ctx context.Context, modulePin bufmoduleref.ModulePin) (bufmodule.Module, error) {
	module, ok := r.moduleIdentityStringToModule[modulePin.IdentityString()]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: modulePin.String(), Err: fs.ErrNotExist}
	}
	return module, nil
}
