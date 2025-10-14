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

package bufpluginv1beta1

import (
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/cmd/buf/internal/command/beta/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver"
)

// NewCommand returns a new Command.
func NewCommand(name string, builder appext.SubCommandBuilder) *appcmd.Command {
	return internal.NewCommand(name, builder, bufcheckserver.V1Beta1Spec)
}
