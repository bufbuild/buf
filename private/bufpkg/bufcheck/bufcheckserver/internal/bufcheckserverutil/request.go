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

package bufcheckserverutil

import (
	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

// Request is a check.Request that also includes bufprotosource functionality.
type Request interface {
	check.Request

	// ProtosourceFiles returns the check.Files as bufprotosource.Files.
	ProtosourceFiles() []bufprotosource.File
	// AgainstProtosourceFiles returns the check.AgainstFiles as bufprotosource.Files.
	AgainstProtosourceFiles() []bufprotosource.File
}

type request struct {
	check.Request

	protosourceFiles        []bufprotosource.File
	againstProtosourceFiles []bufprotosource.File
}

func newRequest(
	checkRequest check.Request,
	protosourceFiles []bufprotosource.File,
	againstProtosourceFiles []bufprotosource.File,
) *request {
	return &request{
		Request:                 checkRequest,
		protosourceFiles:        protosourceFiles,
		againstProtosourceFiles: againstProtosourceFiles,
	}
}

func (r *request) ProtosourceFiles() []bufprotosource.File {
	return r.protosourceFiles
}

func (r *request) AgainstProtosourceFiles() []bufprotosource.File {
	return r.againstProtosourceFiles
}
