// Copyright 2020-2025 Buf Technologies, Inc.
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

package buflsp

// Configuration keys for the LSP.
//
// These are part of the public API of the LSP server, in that any user of
// the LSP can configure it using these keys. The server requests them over
// the LSP protocol as-needed.
//
// Keep in sync with bufbuild/vscode-buf package.json.
const (
	// The strategy for how to calculate the --against input for a breaking
	// check. This must be one of the following values:
	//
	// - "git". Use a particular Git revision to find the against file.
	//
	// - "disk". Use the last-saved value on disk as the against file.
	ConfigBreakingStrategy = "buf.checks.breaking.againstStrategy"
	// The Git revision to use for calculating the --against input for a
	// breaking check when using the "git" strategy.
	ConfigBreakingGitRef = "buf.checks.breaking.againstGitRef"
)

const (
	// Compare against the configured git branch.
	againstGit againstStrategy = iota + 1
	// Against the last saved file on disk (i.e. saved vs unsaved changes).
	againstDisk
)

// againstStrategy is a strategy for selecting which version of a file to use as
// --against for the purposes of breaking lints.
type againstStrategy int

// parseAgainstStrategy parses an againstKind from a config setting sent by
// the client.
//
// Returns againstTrunk, false if the value is not recognized.
func parseAgainstStrategy(s string) (againstStrategy, bool) {
	switch s {
	// These values are the same as those present in the package.json for the
	// VSCode client.
	case "git":
		return againstGit, true
	case "disk":
		return againstDisk, true
	default:
		return againstGit, false
	}
}
