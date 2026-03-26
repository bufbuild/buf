// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufcli

import (
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/spf13/cobra"
)

// errorFormatDescriptions maps error format strings that have non-obvious names
// to a short description for shell completion.
var errorFormatDescriptions = map[string]string{
	"msvs":               "Visual Studio",
	"config-ignore-yaml": "buf.yaml ignore_only snippet",
}

// RegisterFlagCompletionErrorFormat registers shell completion for flags that accept
// a bufanalysis error format value.
func RegisterFlagCompletionErrorFormat(cmd *cobra.Command, flagName string) error {
	return cmd.RegisterFlagCompletionFunc(flagName, cobra.FixedCompletions(
		xslices.Map(bufanalysis.AllFormatStrings, func(s string) string {
			if desc, ok := errorFormatDescriptions[s]; ok {
				return cobra.CompletionWithDesc(s, desc)
			}
			return s
		}),
		cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder,
	))
}

// RegisterFlagCompletionLintErrorFormat registers shell completion for flags that accept
// a lint error format value (which includes the extra "config-ignore-yaml" value).
func RegisterFlagCompletionLintErrorFormat(cmd *cobra.Command, flagName string) error {
	return cmd.RegisterFlagCompletionFunc(flagName, cobra.FixedCompletions(
		xslices.Map(AllLintFormatStrings, func(s string) string {
			if desc, ok := errorFormatDescriptions[s]; ok {
				return cobra.CompletionWithDesc(s, desc)
			}
			return s
		}),
		cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder,
	))
}

// RegisterFlagCompletionOutputFormat registers shell completion for flags that accept
// a bufprint output format value.
func RegisterFlagCompletionOutputFormat(cmd *cobra.Command, flagName string) error {
	return cmd.RegisterFlagCompletionFunc(flagName, cobra.FixedCompletions(
		[]string{
			bufprint.FormatText.String(),
			bufprint.FormatJSON.String(),
		},
		cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder,
	))
}

// RegisterFlagCompletionRuleFormat registers shell completion for flags that accept
// a rule print format value (text or json).
func RegisterFlagCompletionRuleFormat(cmd *cobra.Command, flagName string) error {
	return cmd.RegisterFlagCompletionFunc(flagName, cobra.FixedCompletions(
		AllRuleFormatStrings,
		cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder,
	))
}

// RegisterFlagCompletionVisibility registers shell completion for flags that accept
// a visibility value.
func RegisterFlagCompletionVisibility(cmd *cobra.Command, flagName string) error {
	return cmd.RegisterFlagCompletionFunc(flagName, cobra.FixedCompletions(
		allVisibilityStrings,
		cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder,
	))
}

// RegisterFlagCompletionPluginType registers shell completion for flags that accept
// a plugin type value.
func RegisterFlagCompletionPluginType(cmd *cobra.Command, flagName string) error {
	return cmd.RegisterFlagCompletionFunc(flagName, cobra.FixedCompletions(
		bufplugin.AllPluginTypeStrings,
		cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder,
	))
}

// RegisterFlagCompletionFileVersion registers shell completion for flags that accept
// a bufconfig file version value, ordered with the current recommended version first.
func RegisterFlagCompletionFileVersion(cmd *cobra.Command, flagName string) error {
	return cmd.RegisterFlagCompletionFunc(flagName, cobra.FixedCompletions(
		[]string{
			bufconfig.FileVersionV2.String(),
			bufconfig.FileVersionV1.String(),
			bufconfig.FileVersionV1Beta1.String(),
		},
		cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder,
	))
}
