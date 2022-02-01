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

package buf

import (
	"context"
	"fmt"
	"time"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokencreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokendelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokenget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokenlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/commit/commitget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/commit/commitlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/organization/organizationcreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/organization/organizationdelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/organization/organizationget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/plugincreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/plugindelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/plugindeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/pluginlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/pluginundeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/pluginversion/pluginversionlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorycreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorydelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorydeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositoryget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorylist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositoryundeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/tag/tagcreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/tag/taglist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatecreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatedelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatedeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatelist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templateundeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templateversion/templateversioncreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templateversion/templateversionlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/track/trackdelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/track/tracklist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/breaking"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/build"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/config/configinit"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/config/configlsbreakingrules"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/config/configlslintrules"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/config/configmigratev1beta1"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/export"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/generate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/lint"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/lsfiles"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modclearcache"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modprune"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modupdate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/protoc"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/push"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/registry/registrylogin"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/registry/registrylogout"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
)

const (
	betaConfigDeprecationMessage               = `"buf beta config ..." has been moved to "buf config ...".` + bufcli.DeprecationMessageSuffix
	betaConfigInitDeprecationMessage           = `"buf beta config init" has been moved to "buf config init".` + bufcli.DeprecationMessageSuffix
	betaImageConvertDeprecationMessage         = `"buf beta image convert" has been moved to "buf build".` + bufcli.DeprecationMessageSuffix
	betaModClearCacheDeprecationMessage        = `"buf beta mod clear-cache" has been moved to "buf mod clear-cache".` + bufcli.DeprecationMessageSuffix
	betaModDeprecationMessage                  = `"buf beta mod ..." has been moved to "buf mod ...".` + bufcli.DeprecationMessageSuffix
	betaModExportDeprecationMessage            = `"buf beta mod export" has been moved to "buf export".` + bufcli.DeprecationMessageSuffix
	betaModInitDeprecationMessage              = `"buf beta mod init" has been moved to "buf config init".` + bufcli.DeprecationMessageSuffix
	betaModUpdateDeprecationMessage            = `"buf beta mod update" has been moved to "buf mod update".` + bufcli.DeprecationMessageSuffix
	betaPushDeprecationMessage                 = `"buf beta push" has been moved to "buf mod push".` + bufcli.DeprecationMessageSuffix
	checkBreakingDeprecationMessage            = `"buf check breaking" has been moved to "buf breaking".` + bufcli.DeprecationMessageSuffix
	checkDeprecationMessage                    = `"buf check" sub-commands are now all implemented with the top-level "buf lint" and "buf breaking" commands.` + bufcli.DeprecationMessageSuffix
	checkLintDeprecationMessage                = `"buf check lint" has been moved to "buf lint".` + bufcli.DeprecationMessageSuffix
	checkLsBreakingCheckersDeprecationMessage  = `"buf check ls-breaking-checkers" has been moved to "buf config ls-breaking-rules".` + bufcli.DeprecationMessageSuffix
	checkLsLintCheckersDeprecationMessage      = `"buf check ls-lint-checkers" has been moved to "buf config ls-lint-rules".` + bufcli.DeprecationMessageSuffix
	experimentalDeprecationMessage             = `"buf experimental" sub-commands have moved to "buf beta".` + bufcli.DeprecationMessageSuffix
	experimentalImageConvertDeprecationMessage = `"buf experimental image convert" has been moved to "buf build".` + bufcli.DeprecationMessageSuffix
	imageBuildDeprecationMessage               = `"buf image build" has been moved to "buf build".` + bufcli.DeprecationMessageSuffix
	imageDeprecationMessage                    = `"buf image" sub-commands are now all implemented under the top-level "buf build" command.` + bufcli.DeprecationMessageSuffix
	loginDeprecationMessage                    = `"buf login" has been moved to "buf registry login".` + bufcli.DeprecationMessageSuffix
	logoutDeprecationMessage                   = `"buf logout" has been moved to "buf registry logout".` + bufcli.DeprecationMessageSuffix
	modInitDeprecationMessage                  = `"buf mod init" has been moved to "buf config init".` + bufcli.DeprecationMessageSuffix
	pushDeprecationMessage                     = `"buf push" has been moved to "buf mod push".` + bufcli.DeprecationMessageSuffix
)

// Deprecation messages for the hidden commands replaced by the "completion" command
func shellDeprecationMessage(shell string) string {
	return fmt.Sprintf(`"buf %s-completion" has been moved to "buf completion %s. %s"`, shell, shell, bufcli.DeprecationMessageSuffix)
}

// Main is the entrypoint to the buf CLI.
func Main(name string) {
	appcmd.Main(context.Background(), NewRootCommand(name))
}

// NewRootCommand returns a new root command.
//
// This is public for use in testing.
func NewRootCommand(name string) *appcmd.Command {
	builder := appflag.NewBuilder(
		name,
		appflag.BuilderWithTimeout(120*time.Second),
		appflag.BuilderWithTracing(),
	)
	globalFlags := bufcli.NewGlobalFlags()
	return &appcmd.Command{
		Use:                 name,
		Version:             bufcli.Version,
		BindPersistentFlags: appcmd.BindMultiple(builder.BindRoot, globalFlags.BindRoot),
		SubCommands: []*appcmd.Command{
			build.NewCommand("build", builder),
			export.NewCommand("export", builder),
			lint.NewCommand("lint", builder),
			breaking.NewCommand("breaking", builder),
			generate.NewCommand("generate", builder),
			protoc.NewCommand("protoc", builder),
			lsfiles.NewCommand("ls-files", builder),
			push.NewCommand("push", builder),
			{
				Use:   "mod",
				Short: "Configure and update buf modules.",
				SubCommands: []*appcmd.Command{
					appcmd.NewDeletedCommand("init", modInitDeprecationMessage),
					modprune.NewCommand("prune", builder),
					modupdate.NewCommand("update", builder),
					modclearcache.NewCommand("clear-cache", builder, "cc"),
				},
			},
			{
				Use:   "config",
				Short: "Interact with the configuration of Buf.",
				SubCommands: []*appcmd.Command{
					configinit.NewCommand("init", builder),
					configlslintrules.NewCommand("ls-lint-rules", builder),
					configlsbreakingrules.NewCommand("ls-breaking-rules", builder),
					configmigratev1beta1.NewCommand("migrate-v1beta1", builder),
				},
			},
			{
				Use:   "registry",
				Short: "Interact with the Buf Schema Registry.",
				SubCommands: []*appcmd.Command{
					registrylogin.NewCommand("login", builder),
					registrylogout.NewCommand("logout", builder),
				},
			},
			{
				Use:   "beta",
				Short: "Beta commands. Unstable and will likely change.",
				SubCommands: []*appcmd.Command{
					{
						Use:   "registry",
						Short: "Interact with the Buf Schema Registry.",
						SubCommands: []*appcmd.Command{
							{
								Use:   "organization",
								Short: "Organization commands.",
								SubCommands: []*appcmd.Command{
									organizationcreate.NewCommand("create", builder),
									organizationget.NewCommand("get", builder),
									organizationdelete.NewCommand("delete", builder),
								},
							},
							{
								Use:   "repository",
								Short: "Repository commands.",
								SubCommands: []*appcmd.Command{
									repositorycreate.NewCommand("create", builder),
									repositoryget.NewCommand("get", builder),
									repositorylist.NewCommand("list", builder),
									repositorydelete.NewCommand("delete", builder),
									repositorydeprecate.NewCommand("deprecate", builder),
									repositoryundeprecate.NewCommand("undeprecate", builder),
								},
							},
							{
								Use:    "track",
								Hidden: true, // TODO: (tracks) remove this when ready
								Short:  "Repository track commands.",
								SubCommands: []*appcmd.Command{
									tracklist.NewCommand("list", builder),
									trackdelete.NewCommand("delete", builder),
								},
							},
							{
								Use:   "tag",
								Short: "Repository tag commands.",
								SubCommands: []*appcmd.Command{
									tagcreate.NewCommand("create", builder),
									taglist.NewCommand("list", builder),
								},
							},
							{
								Use:   "commit",
								Short: "Repository commit commands.",
								SubCommands: []*appcmd.Command{
									commitget.NewCommand("get", builder),
									commitlist.NewCommand("list", builder),
								},
							},
							{
								Use:   "plugin",
								Short: "Plugin commands.",
								SubCommands: []*appcmd.Command{
									plugincreate.NewCommand("create", builder),
									pluginlist.NewCommand("list", builder),
									plugindelete.NewCommand("delete", builder),
									plugindeprecate.NewCommand("deprecate", builder),
									pluginundeprecate.NewCommand("undeprecate", builder),
									{
										Use:   "version",
										Short: "Plugin version commands.",
										SubCommands: []*appcmd.Command{
											pluginversionlist.NewCommand("list", builder),
										},
									},
								},
							},
							{
								Use:   "template",
								Short: "Template commands.",
								SubCommands: []*appcmd.Command{
									templatecreate.NewCommand("create", builder),
									templatelist.NewCommand("list", builder),
									templatedelete.NewCommand("delete", builder),
									templatedeprecate.NewCommand("deprecate", builder),
									templateundeprecate.NewCommand("undeprecate", builder),
									{
										Use:   "version",
										Short: "Template version commands.",
										SubCommands: []*appcmd.Command{
											templateversioncreate.NewCommand("create", builder),
											templateversionlist.NewCommand("list", builder),
										},
									},
								},
							},
						},
					},
					{
						Use:        "config",
						Short:      "Interact with the configuration of Buf.",
						Deprecated: betaConfigDeprecationMessage,
						Hidden:     true,
						SubCommands: []*appcmd.Command{
							appcmd.NewDeletedCommand("init", betaConfigInitDeprecationMessage),
						},
					},
					{
						Use:        "image",
						Short:      "Work with Images and FileDescriptorSets.",
						Deprecated: imageDeprecationMessage,
						Hidden:     true,
						SubCommands: []*appcmd.Command{
							appcmd.NewDeletedCommand("convert", betaImageConvertDeprecationMessage),
						},
					},
					{
						Use:        "mod",
						Short:      "Configure and update buf modules.",
						Deprecated: betaModDeprecationMessage,
						Hidden:     true,
						SubCommands: []*appcmd.Command{
							appcmd.NewDeletedCommand("init", betaModInitDeprecationMessage),
							appcmd.NewDeletedCommand("update", betaModUpdateDeprecationMessage),
							appcmd.NewDeletedCommand("export", betaModExportDeprecationMessage),
							appcmd.NewDeletedCommand("clear-cache", betaModClearCacheDeprecationMessage, "cc"),
						},
					},
					appcmd.NewDeletedCommand("push", betaPushDeprecationMessage),
				},
			},
			{
				Use:    "alpha",
				Short:  "Alpha commands. These are so early in development that they should not be used except in development.",
				Hidden: true,
				SubCommands: []*appcmd.Command{
					{
						Use:   "registry",
						Short: "Interact with the Buf Schema Registry.",
						SubCommands: []*appcmd.Command{
							{
								Use:   "token",
								Short: "Token commands.",
								SubCommands: []*appcmd.Command{
									tokencreate.NewCommand("create", builder),
									tokenget.NewCommand("get", builder),
									tokenlist.NewCommand("list", builder),
									tokendelete.NewCommand("delete", builder),
								},
							},
						},
					},
				},
			},
			appcmd.NewDeletedCommand("login", loginDeprecationMessage),
			appcmd.NewDeletedCommand("logout", logoutDeprecationMessage),
			appcmd.NewDeletedCommand("push", pushDeprecationMessage),
			{
				Use:        "image",
				Short:      "Work with Images and FileDescriptorSets.",
				Deprecated: imageDeprecationMessage,
				Hidden:     true,
				SubCommands: []*appcmd.Command{
					appcmd.NewDeletedCommand("build", imageBuildDeprecationMessage),
				},
			},
			{
				Use:        "check",
				Short:      "Run linting or breaking change detection.",
				Deprecated: checkDeprecationMessage,
				Hidden:     true,
				SubCommands: []*appcmd.Command{
					appcmd.NewDeletedCommand("lint", checkLintDeprecationMessage),
					appcmd.NewDeletedCommand("breaking", checkBreakingDeprecationMessage),
					appcmd.NewDeletedCommand("ls-lint-checkers", checkLsLintCheckersDeprecationMessage),
					appcmd.NewDeletedCommand("ls-breaking-checkers", checkLsBreakingCheckersDeprecationMessage),
				},
			},
			{
				Use:        "experimental",
				Short:      "Experimental commands. Unstable and will likely change.",
				Deprecated: experimentalDeprecationMessage,
				Hidden:     true,
				SubCommands: []*appcmd.Command{
					{
						Use:        "image",
						Short:      "Work with Images and FileDescriptorSets.",
						Deprecated: imageDeprecationMessage,
						Hidden:     true,
						SubCommands: []*appcmd.Command{
							appcmd.NewDeletedCommand("convert", experimentalImageConvertDeprecationMessage),
						},
					},
				},
			},
			// Deprecated shell completion commands (replaced by "buf completion")
			{
				Use:        "bash-completion",
				Deprecated: shellDeprecationMessage("bash"),
				Hidden:     true,
				Run:        appcmd.NoAction,
			},
			{
				Use:        "fish-completion",
				Deprecated: shellDeprecationMessage("fish"),
				Hidden:     true,
				Run:        appcmd.NoAction,
			},
			{
				Use:        "zsh-completion",
				Deprecated: shellDeprecationMessage("zsh"),
				Hidden:     true,
				Run:        appcmd.NoAction,
			},
		},
	}
}
