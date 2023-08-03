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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
)

// ErrModuleDoesNotExist is an error returned when looking for a remote module.
var ErrModuleDoesNotExist = errors.New("BSR module does not exist")

// ReadModuleErrorCode is the type of errors that can be thrown by the syncer when reading a module
// from a passed module directory.
type ReadModuleErrorCode int

const (
	// ReadModuleErrorCodeUnknown is any unknown or unexpected error while reading a module.
	ReadModuleErrorCodeUnknown = iota
	// ReadModuleErrorCodeModuleNotFound happens when the passed module directory does not have any
	// module.
	ReadModuleErrorCodeModuleNotFound
	// ReadModuleErrorCodeUnnamedModule happens when the read module does not have a name.
	ReadModuleErrorCodeUnnamedModule
	// ReadModuleErrorCodeInvalidModuleConfig happens when the module directory has an invalid module
	// configuration.
	ReadModuleErrorCodeInvalidModuleConfig
	// ReadModuleErrorCodeBuildModule happens when the read module errors building.
	ReadModuleErrorCodeBuildModule
	// ReadModuleErrorCodeNameDifferentThanHEAD happens when the read module has a different name than
	// the module name in the branch HEAD commit.
	ReadModuleErrorCodeNameDifferentThanHEAD
)

// ReadModuleError is an error that happens when trying to read a module from a module directory in
// a git commit.
type ReadModuleError struct {
	err       error
	code      ReadModuleErrorCode
	branch    string
	commit    string
	moduleDir string
}

func (e *ReadModuleError) Error() string {
	return fmt.Sprintf(
		"read module in branch %s, commit %s, directory %s: %s",
		e.branch, e.commit, e.moduleDir, e.err.Error(),
	)
}

// ErrorHandler handles errors reported by the Syncer before or during the sync process.
type ErrorHandler interface {
	// StopLookback is invoked when deciding on a git start sync point.
	//
	// For each branch to be synced, the Syncer travels back from HEAD looking for modules in the
	// given module directories, until finding a commit that is already synced to the BSR, or the
	// beginning of the Git repository.
	//
	// The syncer might find errors trying to read a module in that directory. Those errors are sent
	// to this function to decide if the Syncer should stop looking back or not, and choose the
	// previous one (if any) as the start sync point.
	//
	// e.g.: The git commits in topological order are: `a -> ... -> z (HEAD)`, and the modules on a
	// given module directory are:
	//
	// commit | module name or failure | could be synced? | why?
	// ----------------------------------------------------------------------------------------
	// z      | buf.build/acme/foo     | Y                | HEAD
	// y      | buf.build/acme/foo     | Y                | same as HEAD
	// x      | buf.build/acme/bar     | N                | different than HEAD
	// w      | unnamed module         | N                | no module name
	// v      | unbuildable module     | N                | module does not build
	// u      | module not found       | N                | no module name, no 'buf.yaml' file
	// t      | buf.build/acme/foo     | Y                | same as HEAD
	// s      | buf.build/acme/foo     | Y                | same as HEAD
	// r      | buf.build/acme/foo     | N                | already synced to the BSR
	//
	// If this func returns 'true' for any `ReadModuleErrorCode`, then the syncer will stop looking
	// when reaching the commit `r` because it already exists in the BSR, select `s` as the start sync
	// point, and the synced commits into the BSR will be [s, t, x, y, z].
	//
	// On the other hand, if this func returns true for `ReadModuleErrorCodeModuleNotFound`, the
	// syncer will stop looking when reaching the commit `u`, will select `v` as the start sync point,
	// and the synced commits into the BSR will be [x, y, z].
	StopLookback(err *ReadModuleError) bool
	// InvalidSyncPoint is invoked by Syncer upon encountering a module's branch sync point that is
	// invalid. A typical example is either a sync point that points to a commit that cannot be found
	// anymore, or the commit itself has been corrupted.
	//
	// Returning an error will abort sync.
	InvalidSyncPoint(
		module bufmoduleref.ModuleIdentity,
		branch string,
		syncPoint git.Hash,
		isGitDefaultBranch bool,
		err error,
	) error
}
