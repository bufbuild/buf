// Copyright 2020-2021 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/pluginlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/pluginversion/pluginversionlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorycreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorydelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositoryget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorylist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/tag/tagcreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/tag/taglist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatecreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatedelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatelist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templateversion/templateversioncreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templateversion/templateversionlist"
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
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modpush"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modupdate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/protoc"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/registry/registrylogin"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/registry/registrylogout"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/version"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
)

const (
	betaConfigDeprecationMessage               = `"buf beta config ..." has been moved to "buf config ...".`
	betaConfigInitDeprecationMessage           = `"buf beta config init" has been moved to "buf config init".`
	betaImageConvertDeprecationMessage         = `"buf beta image convert" has been moved to "buf build".`
	betaModClearCacheDeprecationMessage        = `"buf beta mod clear-cache" has been moved to "buf mod clear-cache".`
	betaModDeprecationMessage                  = `"buf beta mod ..." has been moved to "buf mod ...".`
	betaModExportDeprecationMessage            = `"buf beta mod export" has been moved to "buf export".`
	betaModInitDeprecationMessage              = `"buf beta mod init" has been moved to "buf config init".`
	betaModUpdateDeprecationMessage            = `"buf beta mod update" has been moved to "buf mod update".`
	betaPushDeprecationMessage                 = `"buf beta push" has been moved to "buf mod push".`
	checkBreakingDeprecationMessage            = `"buf check breaking" has been moved to "buf breaking".`
	checkDeprecationMessage                    = `"buf check" sub-commands are now all implemented with the top-level "buf lint" and "buf breaking" commands.`
	checkLintDeprecationMessage                = `"buf check lint" has been moved to "buf lint".`
	checkLsBreakingCheckersDeprecationMessage  = `"buf check ls-breaking-checkers" has been moved to "buf config ls-breaking-rules".`
	checkLsLintCheckersDeprecationMessage      = `"buf check ls-lint-checkers" has been moved to "buf config ls-lint-rules".`
	experimentalDeprecationMessage             = `"buf experimental" sub-commands have moved to "buf beta".`
	experimentalImageConvertDeprecationMessage = `"buf experimental image convert" has been moved to "buf build".`
	imageBuildDeprecationMessage               = `"buf image build" has been moved to "buf build".`
	imageDeprecationMessage                    = `"buf image" sub-commands are now all implemented under the top-level "buf build" command.`
	loginDeprecationMessage                    = `"buf login" has been moved to "buf registry login".`
	logoutDeprecationMessage                   = `"buf logout" has been moved to "buf registry logout".`
	modInitDeprecationMessage                  = `"buf mod init" has been moved to "buf config init".`
	pushDeprecationMessage                     = `"buf push" has been moved to "buf mod push".`
)

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
		Use: name,
		SubCommands: []*appcmd.Command{
			build.NewCommand("build", builder),
			export.NewCommand("export", builder),
			lint.NewCommand("lint", builder),
			breaking.NewCommand("breaking", builder),
			generate.NewCommand("generate", builder),
			protoc.NewCommand("protoc", builder),
			lsfiles.NewCommand("ls-files", builder),
			version.NewCommand("version", builder),
			{
				Use:   "mod",
				Short: "Configure and update buf modules.",
				SubCommands: []*appcmd.Command{
					appcmd.NewDeletedCommand("init", modInitDeprecationMessage),
					modprune.NewCommand("prune", builder),
					modpush.NewCommand("push", builder),
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
								},
							},
							//{
							//	Use:   "branch",
							//	Short: "Repository branch commands.",
							//	SubCommands: []*appcmd.Command{
							//		branchcreate.NewCommand("create", builder),
							//		branchlist.NewCommand("list", builder),
							//	},
							//},
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
		},
		BindPersistentFlags: appcmd.BindMultiple(builder.BindRoot, globalFlags.BindRoot),
	}
}
