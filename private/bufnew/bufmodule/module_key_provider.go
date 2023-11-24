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

package bufmodule

import (
	"context"
)

// ModuleKeyProvider provides ModuleKeys for ModuleRefs.
type ModuleKeyProvider interface {
	// GetModuleKeysForModuleRefs gets the ModuleKeys for the given ModuleRefs.
	//
	// Resolution of the ModuleRefs is done per the ModuleRef documentation.
	//
	// If there is no error, the length of the ModuleKeys returned will match the length of the ModuleRefs.
	// If there is an error, no ModuleKeys will be returned.
	// An error with fs.ErrNotExist will be returned if a ModuleRef is not found.
	//
	// All ModuleKeys will have CommitID set.
	GetModuleKeysForModuleRefs(context.Context, ...ModuleRef) ([]ModuleKey, error)
}
