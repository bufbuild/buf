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

package bufprotopluginexec

import (
	"errors"
	"fmt"
	"os"
)

// handlePotentialTooManyFilesError checks if the error is a result of too many files
// being open, and if so, modifies the output error with a help message.
//
// This could potentially go in osext, but we want to provide a specific help
// message referencing StrategyAll, so it is simplest to just put this here for now.
func handlePotentialTooManyFilesError(err error) error {
	if isTooManyFilesError(err) {
		return fmt.Errorf("%w: %s", err, tooManyFilesHelpMessage)
	}
	return err
}

func isTooManyFilesError(err error) bool {
	var syscallError *os.SyscallError
	if errors.As(err, &syscallError) {
		// This may not actually be correct on other platforms, however the worst case
		// is that we just don't provide the additional help message.
		//
		// Note that syscallError.Syscall has both equalled "pipe" and "fork/exec" in testing, but
		// we don't match on this as this could be particularly prone to being platform-specific.
		return syscallError.Err != nil && syscallError.Err.Error() == "too many open files"
	}
	return false
}
