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

package buf

import (
	"context"
	"time"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/package/goversion"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/package/mavenversion"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/package/npmversion"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/package/pythonversion"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/package/swiftversion"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/protoc"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokendelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokenget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/registry/token/tokenlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/repo/reposync"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/alpha/workspace/workspacepush"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/graph"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/migratev1beta1"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/price"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/commit/commitget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/commit/commitlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/draft/draftdelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/draft/draftlist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/organization/organizationcreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/organization/organizationdelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/organization/organizationget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/plugindelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/plugin/pluginpush"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorycreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorydelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorydeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositoryget"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositorylist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositoryundeprecate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/repository/repositoryupdate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/tag/tagcreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/tag/taglist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/webhook/webhookcreate"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/webhook/webhookdelete"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/webhook/webhooklist"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/stats"
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
	return &appcmd.Command{
		Use:                 name,
		Short:               "The Buf CLI",
		Long:                "A tool for working with Protocol Buffers and managing resources on the Buf Schema Registry (BSR)",
		Version:             bufcli.Version,
		BindPersistentFlags: builder.BindRoot,
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
					graph.NewCommand("graph", builder),
					price.NewCommand("price", builder),
					stats.NewCommand("stats", builder),
					migratev1beta1.NewCommand("migrate-v1beta1", builder),
					studioagent.NewCommand("studio-agent", builder),
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
								Use:   "webhook",
								Short: "Manage webhooks for a repository on the Buf Schema Registry",
								SubCommands: []*appcmd.Command{
									webhookcreate.NewCommand("create", builder),
									webhookdelete.NewCommand("delete", builder),
									webhooklist.NewCommand("list", builder),
								},
							},
							{
								Use:   "plugin",
								Short: "Manage plugins on the Buf Schema Registry",
								SubCommands: []*appcmd.Command{
									pluginpush.NewCommand("push", builder),
									plugindelete.NewCommand("delete", builder),
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
					{
						Use:   "registry",
						Short: "Manage assets on the Buf Schema Registry",
						SubCommands: []*appcmd.Command{
							{
								Use:   "token",
								Short: "Manage user tokens",
								SubCommands: []*appcmd.Command{
									tokenget.NewCommand("get", builder),
									tokenlist.NewCommand("list", builder),
									tokendelete.NewCommand("delete", builder),
								},
							},
						},
					},
					{
						Use:   "sdk",
						Short: "Manage Generated SDKs",
						SubCommands: []*appcmd.Command{
							goversion.NewCommand("go-version", builder),
							mavenversion.NewCommand("maven-version", builder),
							npmversion.NewCommand("npm-version", builder),
							swiftversion.NewCommand("swift-version", builder),
							pythonversion.NewCommand("python-version", builder),
						},
					},
					{
						Use:   "repo",
						Short: "Manage Git repositories",
						SubCommands: []*appcmd.Command{
							reposync.NewCommand("sync", builder),
						},
					},
					{
						Use:   "workspace",
						Short: "Manage workspaces",
						SubCommands: []*appcmd.Command{
							workspacepush.NewCommand("push", builder),
						},
					},
				},
			},
		},
	}
}
