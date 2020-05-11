// Copyright 2020 Buf Technologies Inc.
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

package buf

import (
	"context"

	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
)

const version = "0.12.0"

// Main is the main.
func Main(use string, options ...RootCommandOption) {
	appcmd.Main(context.Background(), newRootCommand(use, options...), version)
}

// RootCommandOption is an option for a root Command.
type RootCommandOption func(*appcmd.Command, appflag.Builder)
