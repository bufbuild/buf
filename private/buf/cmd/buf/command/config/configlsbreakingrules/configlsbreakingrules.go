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

package configlsbreakingrules

import (
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/config/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	return internal.NewLSCommand(
		name,
		builder,
		"breaking",
		bufbreaking.GetAllRulesV1Beta1,
		bufbreaking.GetAllRulesV1,
		bufbreaking.GetAllRulesV2,
		func(moduleConfig bufconfig.ModuleConfig) ([]bufcheck.Rule, error) {
			return bufbreaking.RulesForConfig(moduleConfig.BreakingConfig())
		},
	)
}
