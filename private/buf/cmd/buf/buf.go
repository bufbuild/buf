// Copyright 2020-2023 Buf Technologies, Inc.
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
	curatedplugindelete "github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/plugin/plugindelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/plugin/pluginpush"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/protoc"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokencreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokendelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokenget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokenlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/stats"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/migratev1beta1"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/commit/commitget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/commit/commitlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/draft/draftdelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/draft/draftlist"
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
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositoryupdate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/tag/tagcreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/tag/taglist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatecreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatedelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatedeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templatelist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templateundeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templateversion/templateversioncreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/template/templateversion/templateversionlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/webhook/webhookcreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/webhook/webhookdelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/webhook/webhooklist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/studioagent"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/breaking"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/build"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/convert"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/curl"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/export"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/format"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/generate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/lint"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/lsfiles"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modclearcache"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modinit"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modlsbreakingrules"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modlslintrules"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modopen"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modprune"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/modupdate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/push"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/registry/registrylogin"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/registry/registrylogout"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
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
	noTimeoutBuilder := appflag.NewBuilder(
		name,
		appflag.BuilderWithTracing(),
	)
	globalFlags := bufcli.NewGlobalFlags()
	return &appcmd.Command{
		Use:                 name,
		Short:               "The Buf CLI",
		Long:                "A tool for working with Protocol Buffers and managing resources on the Buf Schema Registry (BSR)",
		Version:             bufcli.Version,
		BindPersistentFlags: appcmd.BindMultiple(builder.BindRoot, globalFlags.BindRoot),
		SubCommands: []*appcmd.Command{
			build.NewCommand("build", builder),
			export.NewCommand("export", builder),
			format.NewCommand("format", builder),
			lint.NewCommand("lint", builder),
			breaking.NewCommand("breaking", builder),
			generate.NewCommand("generate", builder),
			lsfiles.NewCommand("ls-files", builder),
			push.NewCommand("push", builder),
			convert.NewCommand("convert", builder),
			curl.NewCommand("curl", builder),
			{
				Use:   "mod",
				Short: "Manage Buf modules",
				SubCommands: []*appcmd.Command{
					modinit.NewCommand("init", builder),
					modprune.NewCommand("prune", builder),
					modupdate.NewCommand("update", builder),
					modopen.NewCommand("open", builder),
					modclearcache.NewCommand("clear-cache", builder, "cc"),
					modlslintrules.NewCommand("ls-lint-rules", builder),
					modlsbreakingrules.NewCommand("ls-breaking-rules", builder),
				},
			},
			{
				Use:   "registry",
				Short: "Manage assets on the Buf Schema Registry",
				SubCommands: []*appcmd.Command{
					registrylogin.NewCommand("login", builder),
					registrylogout.NewCommand("logout", builder),
				},
			},
			{
				Use:   "beta",
				Short: "Beta commands. Unstable and likely to change",
				SubCommands: []*appcmd.Command{
					migratev1beta1.NewCommand("migrate-v1beta1", builder),
					studioagent.NewCommand("studio-agent", noTimeoutBuilder),
					{
						Use:   "registry",
						Short: "Manage assets on the Buf Schema Registry",
						SubCommands: []*appcmd.Command{
							{
								Use:   "organization",
								Short: "Manage organizations",
								SubCommands: []*appcmd.Command{
									organizationcreate.NewCommand("create", builder),
									organizationget.NewCommand("get", builder),
									organizationdelete.NewCommand("delete", builder),
								},
							},
							{
								Use:   "repository",
								Short: "Manage repositories",
								SubCommands: []*appcmd.Command{
									repositorycreate.NewCommand("create", builder),
									repositoryget.NewCommand("get", builder),
									repositorylist.NewCommand("list", builder),
									repositorydelete.NewCommand("delete", builder),
									repositorydeprecate.NewCommand("deprecate", builder),
									repositoryundeprecate.NewCommand("undeprecate", builder),
									repositoryupdate.NewCommand("update", builder),
								},
							},
							{
								Use:   "tag",
								Short: "Manage a repository's tags",
								SubCommands: []*appcmd.Command{
									tagcreate.NewCommand("create", builder),
									taglist.NewCommand("list", builder),
								},
							},
							{
								Use:   "commit",
								Short: "Manage a repository's commits",
								SubCommands: []*appcmd.Command{
									commitget.NewCommand("get", builder),
									commitlist.NewCommand("list", builder),
								},
							},
							{
								Use:   "draft",
								Short: "Manage a repository's drafts",
								SubCommands: []*appcmd.Command{
									draftdelete.NewCommand("delete", builder),
									draftlist.NewCommand("list", builder),
								},
							},
							{
								Use:   "plugin",
								Short: "Manage Protobuf plugins",
								SubCommands: []*appcmd.Command{
									plugincreate.NewCommand("create", builder),
									pluginlist.NewCommand("list", builder),
									plugindelete.NewCommand("delete", builder),
									plugindeprecate.NewCommand("deprecate", builder),
									pluginundeprecate.NewCommand("undeprecate", builder),
									{
										Use:   "version",
										Short: "Manage Protobuf plugin versions",
										SubCommands: []*appcmd.Command{
											pluginversionlist.NewCommand("list", builder),
										},
									},
								},
							},
							{
								Use:   "template",
								Short: "Manage Protobuf templates on the Buf Schema Registry",
								SubCommands: []*appcmd.Command{
									templatecreate.NewCommand("create", builder),
									templatelist.NewCommand("list", builder),
									templatedelete.NewCommand("delete", builder),
									templatedeprecate.NewCommand("deprecate", builder),
									templateundeprecate.NewCommand("undeprecate", builder),
									{
										Use:   "version",
										Short: "Manage Protobuf template versions",
										SubCommands: []*appcmd.Command{
											templateversioncreate.NewCommand("create", builder),
											templateversionlist.NewCommand("list", builder),
										},
									},
								},
							},
							{
								Use:   "webhook",
								Short: "Manage webhooks for a repository on the Buf Schema Registry",
								SubCommands: []*appcmd.Command{
									webhookcreate.NewCommand("create", builder),
									webhookdelete.NewCommand("delete", builder),
									webhooklist.NewCommand("list", builder),
								},
							},
						},
					},
				},
			},
			{
				Use:    "alpha",
				Short:  "Alpha commands. Unstable and recommended only for experimentation. These may be deleted",
				Hidden: true,
				SubCommands: []*appcmd.Command{
					protoc.NewCommand("protoc", builder),
					stats.NewCommand("stats", builder),
					{
						Use:   "registry",
						Short: "Manage assets on the Buf Schema Registry",
						SubCommands: []*appcmd.Command{
							{
								Use:   "token",
								Short: "Manage user tokens",
								SubCommands: []*appcmd.Command{
									tokencreate.NewCommand("create", builder),
									tokenget.NewCommand("get", builder),
									tokenlist.NewCommand("list", builder),
									tokendelete.NewCommand("delete", builder),
								},
							},
						},
					},
					{
						Use:   "plugin",
						Short: "Manage plugins on the Buf Schema Registry",
						SubCommands: []*appcmd.Command{
							pluginpush.NewCommand("push", builder),
							curatedplugindelete.NewCommand("delete", builder),
						},
					},
				},
			},
		},
	}
}
