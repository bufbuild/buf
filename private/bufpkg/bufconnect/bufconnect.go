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

// Package bufconnect provides buf-specific Connect functionality.
package bufconnect

const (
	// AuthenticationHeader is the standard OAuth header used for authenticating
	// a user. Ignore the misnomer.
	AuthenticationHeader = "Authorization"
	// AuthenticationTokenPrefix is the standard OAuth token prefix.
	// We use it for familiarity.
	AuthenticationTokenPrefix = "Bearer "
	// CliVersionHeaderName is the name of the header carrying the buf CLI version.
	CliVersionHeaderName = "buf-version"
	// CLIWarningHeaderName is the name of the header carrying a base64-encoded warning message
	// from the server to the CLI.
	CLIWarningHeaderName = "buf-warning-bin"
	// DefaultRemote is the default remote if none can be inferred from a module name.
	DefaultRemote = "buf.build"
)
