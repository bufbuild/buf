package completion

// Copyright 2020-2022 Buf Technologies, Inc.
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

import (
	"errors"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/spf13/cobra"
)

var supportedShells = []string{"bash", "fish", "powershell", "zsh"}

func NewCommand(cmd *cobra.Command, container app.Container) *cobra.Command {
	return &cobra.Command{
		Use:   "completion <" + strings.Join(supportedShells, "|") + ">",
		Short: "Generate auto-completion scripts for commonly used shells.",
		Long:  "Output shell completion code for the specified shell to stdout.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			shell := args[0]
			switch shell {
			case "bash":
				return cmd.GenBashCompletion(container.Stdout())
			case "fish":
				return cmd.GenFishCompletion(container.Stdout(), true)
			case "powershell":
				return cmd.GenPowerShellCompletion(container.Stdout())
			case "zsh":
				return cmd.GenZshCompletion(container.Stdout())
			default:
				return errors.New("unrecognized shell")
			}
		},
	}
}
