
# buf {#buf}
The Buf CLI

### Usage {#buf-usage} 
```terminal
$ buf [flags]
```

### Description {#buf-description}

A tool for working with Protocol Buffers and managing resources on the Buf Schema Registry (BSR)
 

### Flags {#buf-flags}

```
      --debug               Turn on debug logging
  -h, --help                help for buf
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
      --version             Print the version
```

### Subcommands {#buf-subcommands}

* [buf beta](#buf-beta)	 - Beta commands. Unstable and likely to change
* [buf breaking](#buf-breaking)	 - Verify no breaking changes have been made
* [buf build](#buf-build)	 - Build Protobuf files into a Buf image
* [buf convert](#buf-convert)	 - Convert a message from binary to JSON or vice versa
* [buf curl](#buf-curl)	 - Invoke an RPC endpoint, a la 'cURL'
* [buf export](#buf-export)	 - Export proto files from one location to another
* [buf format](#buf-format)	 - Format Protobuf files
* [buf generate](#buf-generate)	 - Generate code with protoc plugins
* [buf lint](#buf-lint)	 - Run linting on Protobuf files
* [buf mod](#buf-mod)	 - Manage Buf modules
* [buf push](#buf-push)	 - Push a module to a registry
* [buf registry](#buf-registry)	 - Manage assets on the Buf Schema Registry


# buf beta {#buf-beta}
Beta commands. Unstable and likely to change

### Usage {#buf-beta-usage} 
```terminal
$ buf beta [flags]
```

### Flags {#buf-beta-flags}

```
  -h, --help   help for beta
```

### Flags inherited from parent commands {#buf-beta-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-subcommands}

* [buf beta migrate-v1beta1](#buf-beta-migrate-v1beta1)	 - Migrate v1beta1 configuration to the latest version
* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry
* [buf beta studio-agent](#buf-beta-studio-agent)	 - Run an HTTP(S) server as the Studio agent

### Parent Command {#buf-beta-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf beta migrate-v1beta1 {#buf-beta-migrate-v1beta1}
Migrate v1beta1 configuration to the latest version

### Usage {#buf-beta-migrate-v1beta1-usage} 
```terminal
$ buf beta migrate-v1beta1 <directory> [flags]
```

### Description {#buf-beta-migrate-v1beta1-description}

Migrate any v1beta1 configuration files in the directory to the latest version.
Defaults to the current directory if not specified.
 

### Flags {#buf-beta-migrate-v1beta1-flags}

```
  -h, --help   help for migrate-v1beta1
```

### Flags inherited from parent commands {#buf-beta-migrate-v1beta1-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-migrate-v1beta1-parent-command}

* [buf beta](#buf-beta)	 - Beta commands. Unstable and likely to change

# buf beta registry {#buf-beta-registry}
Manage assets on the Buf Schema Registry

### Usage {#buf-beta-registry-usage} 
```terminal
$ buf beta registry [flags]
```

### Flags {#buf-beta-registry-flags}

```
  -h, --help   help for registry
```

### Flags inherited from parent commands {#buf-beta-registry-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-subcommands}

* [buf beta registry commit](#buf-beta-registry-commit)	 - Manage a repository's commits
* [buf beta registry draft](#buf-beta-registry-draft)	 - Manage a repository's drafts
* [buf beta registry organization](#buf-beta-registry-organization)	 - Manage organizations
* [buf beta registry plugin](#buf-beta-registry-plugin)	 - Manage Protobuf plugins
* [buf beta registry repository](#buf-beta-registry-repository)	 - Manage repositories
* [buf beta registry tag](#buf-beta-registry-tag)	 - Manage a repository's tags
* [buf beta registry template](#buf-beta-registry-template)	 - Manage Protobuf templates on the Buf Schema Registry
* [buf beta registry webhook](#buf-beta-registry-webhook)	 - Manage webhooks for a repository on the Buf Schema Registry

### Parent Command {#buf-beta-registry-parent-command}

* [buf beta](#buf-beta)	 - Beta commands. Unstable and likely to change

# buf beta registry commit {#buf-beta-registry-commit}
Manage a repository's commits

### Usage {#buf-beta-registry-commit-usage} 
```terminal
$ buf beta registry commit [flags]
```

### Flags {#buf-beta-registry-commit-flags}

```
  -h, --help   help for commit
```

### Flags inherited from parent commands {#buf-beta-registry-commit-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-commit-subcommands}

* [buf beta registry commit get](#buf-beta-registry-commit-get)	 - Get commit details
* [buf beta registry commit list](#buf-beta-registry-commit-list)	 - List repository commits

### Parent Command {#buf-beta-registry-commit-parent-command}

* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry

# buf beta registry commit get {#buf-beta-registry-commit-get}
Get commit details

### Usage {#buf-beta-registry-commit-get-usage} 
```terminal
$ buf beta registry commit get <buf.build/owner/repository[:ref]> [flags]
```

### Flags {#buf-beta-registry-commit-get-flags}

```
      --format string   The output format to use. Must be one of [text,json] (default "text")
  -h, --help            help for get
```

### Flags inherited from parent commands {#buf-beta-registry-commit-get-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-commit-get-parent-command}

* [buf beta registry commit](#buf-beta-registry-commit)	 - Manage a repository's commits

# buf beta registry commit list {#buf-beta-registry-commit-list}
List repository commits

### Usage {#buf-beta-registry-commit-list-usage} 
```terminal
$ buf beta registry commit list <buf.build/owner/repository[:ref]> [flags]
```

### Flags {#buf-beta-registry-commit-list-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for list
      --page-size uint32    The page size (default 10)
      --page-token string   The page token. If more results are available, a "next_page" key is present in the --format=json output
      --reverse             Reverse the results
```

### Flags inherited from parent commands {#buf-beta-registry-commit-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-commit-list-parent-command}

* [buf beta registry commit](#buf-beta-registry-commit)	 - Manage a repository's commits

# buf beta registry draft {#buf-beta-registry-draft}
Manage a repository's drafts

### Usage {#buf-beta-registry-draft-usage} 
```terminal
$ buf beta registry draft [flags]
```

### Flags {#buf-beta-registry-draft-flags}

```
  -h, --help   help for draft
```

### Flags inherited from parent commands {#buf-beta-registry-draft-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-draft-subcommands}

* [buf beta registry draft delete](#buf-beta-registry-draft-delete)	 - Delete a repository draft
* [buf beta registry draft list](#buf-beta-registry-draft-list)	 - List repository drafts

### Parent Command {#buf-beta-registry-draft-parent-command}

* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry

# buf beta registry draft delete {#buf-beta-registry-draft-delete}
Delete a repository draft

### Usage {#buf-beta-registry-draft-delete-usage} 
```terminal
$ buf beta registry draft delete <buf.build/owner/repository:draft> [flags]
```

### Flags {#buf-beta-registry-draft-delete-flags}

```
      --force   Force deletion without confirming. Use with caution
  -h, --help    help for delete
```

### Flags inherited from parent commands {#buf-beta-registry-draft-delete-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-draft-delete-parent-command}

* [buf beta registry draft](#buf-beta-registry-draft)	 - Manage a repository's drafts

# buf beta registry draft list {#buf-beta-registry-draft-list}
List repository drafts

### Usage {#buf-beta-registry-draft-list-usage} 
```terminal
$ buf beta registry draft list <buf.build/owner/repository> [flags]
```

### Flags {#buf-beta-registry-draft-list-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for list
      --page-size uint32    The page size (default 10)
      --page-token string   The page token. If more results are available, a "next_page" key is present in the --format=json output
      --reverse             Reverse the results
```

### Flags inherited from parent commands {#buf-beta-registry-draft-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-draft-list-parent-command}

* [buf beta registry draft](#buf-beta-registry-draft)	 - Manage a repository's drafts

# buf beta registry organization {#buf-beta-registry-organization}
Manage organizations

### Usage {#buf-beta-registry-organization-usage} 
```terminal
$ buf beta registry organization [flags]
```

### Flags {#buf-beta-registry-organization-flags}

```
  -h, --help   help for organization
```

### Flags inherited from parent commands {#buf-beta-registry-organization-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-organization-subcommands}

* [buf beta registry organization create](#buf-beta-registry-organization-create)	 - Create a new BSR organization
* [buf beta registry organization delete](#buf-beta-registry-organization-delete)	 - Delete a BSR organization
* [buf beta registry organization get](#buf-beta-registry-organization-get)	 - Get a BSR organization

### Parent Command {#buf-beta-registry-organization-parent-command}

* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry

# buf beta registry organization create {#buf-beta-registry-organization-create}
Create a new BSR organization

### Usage {#buf-beta-registry-organization-create-usage} 
```terminal
$ buf beta registry organization create <buf.build/organization> [flags]
```

### Flags {#buf-beta-registry-organization-create-flags}

```
      --format string   The output format to use. Must be one of [text,json] (default "text")
  -h, --help            help for create
```

### Flags inherited from parent commands {#buf-beta-registry-organization-create-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-organization-create-parent-command}

* [buf beta registry organization](#buf-beta-registry-organization)	 - Manage organizations

# buf beta registry organization delete {#buf-beta-registry-organization-delete}
Delete a BSR organization

### Usage {#buf-beta-registry-organization-delete-usage} 
```terminal
$ buf beta registry organization delete <buf.build/organization> [flags]
```

### Flags {#buf-beta-registry-organization-delete-flags}

```
      --force   Force deletion without confirming. Use with caution
  -h, --help    help for delete
```

### Flags inherited from parent commands {#buf-beta-registry-organization-delete-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-organization-delete-parent-command}

* [buf beta registry organization](#buf-beta-registry-organization)	 - Manage organizations

# buf beta registry organization get {#buf-beta-registry-organization-get}
Get a BSR organization

### Usage {#buf-beta-registry-organization-get-usage} 
```terminal
$ buf beta registry organization get <buf.build/organization> [flags]
```

### Flags {#buf-beta-registry-organization-get-flags}

```
      --format string   The output format to use. Must be one of [text,json] (default "text")
  -h, --help            help for get
```

### Flags inherited from parent commands {#buf-beta-registry-organization-get-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-organization-get-parent-command}

* [buf beta registry organization](#buf-beta-registry-organization)	 - Manage organizations

# buf beta registry plugin {#buf-beta-registry-plugin}
Manage Protobuf plugins

### Usage {#buf-beta-registry-plugin-usage} 
```terminal
$ buf beta registry plugin [flags]
```

### Flags {#buf-beta-registry-plugin-flags}

```
  -h, --help   help for plugin
```

### Flags inherited from parent commands {#buf-beta-registry-plugin-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-plugin-subcommands}

* [buf beta registry plugin create](#buf-beta-registry-plugin-create)	 - Create a Protobuf plugin
* [buf beta registry plugin delete](#buf-beta-registry-plugin-delete)	 - Delete a Protobuf plugin
* [buf beta registry plugin deprecate](#buf-beta-registry-plugin-deprecate)	 - Deprecate a Protobuf plugin
* [buf beta registry plugin list](#buf-beta-registry-plugin-list)	 - List plugins on the specified BSR
* [buf beta registry plugin undeprecate](#buf-beta-registry-plugin-undeprecate)	 - Undeprecate a plugin
* [buf beta registry plugin version](#buf-beta-registry-plugin-version)	 - Manage Protobuf plugin versions

### Parent Command {#buf-beta-registry-plugin-parent-command}

* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry

# buf beta registry plugin create {#buf-beta-registry-plugin-create}
Create a Protobuf plugin

### Usage {#buf-beta-registry-plugin-create-usage} 
```terminal
$ buf beta registry plugin create <buf.build/owner/plugins/plugin> [flags]
```

### Flags {#buf-beta-registry-plugin-create-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for create
      --visibility string   The plugin's visibility setting. Must be one of [public,private]
```

### Flags inherited from parent commands {#buf-beta-registry-plugin-create-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-plugin-create-parent-command}

* [buf beta registry plugin](#buf-beta-registry-plugin)	 - Manage Protobuf plugins

# buf beta registry plugin delete {#buf-beta-registry-plugin-delete}
Delete a Protobuf plugin

### Usage {#buf-beta-registry-plugin-delete-usage} 
```terminal
$ buf beta registry plugin delete <buf.build/owner/plugins/plugin> [flags]
```

### Flags {#buf-beta-registry-plugin-delete-flags}

```
      --force   Force deletion without confirming. Use with caution
  -h, --help    help for delete
```

### Flags inherited from parent commands {#buf-beta-registry-plugin-delete-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-plugin-delete-parent-command}

* [buf beta registry plugin](#buf-beta-registry-plugin)	 - Manage Protobuf plugins

# buf beta registry plugin deprecate {#buf-beta-registry-plugin-deprecate}
Deprecate a Protobuf plugin

### Usage {#buf-beta-registry-plugin-deprecate-usage} 
```terminal
$ buf beta registry plugin deprecate <buf.build/owner/plugins/plugin> [flags]
```

### Flags {#buf-beta-registry-plugin-deprecate-flags}

```
  -h, --help             help for deprecate
      --message string   The message to display with deprecation warnings
```

### Flags inherited from parent commands {#buf-beta-registry-plugin-deprecate-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-plugin-deprecate-parent-command}

* [buf beta registry plugin](#buf-beta-registry-plugin)	 - Manage Protobuf plugins

# buf beta registry plugin list {#buf-beta-registry-plugin-list}
List plugins on the specified BSR

### Usage {#buf-beta-registry-plugin-list-usage} 
```terminal
$ buf beta registry plugin list <buf.build> [flags]
```

### Flags {#buf-beta-registry-plugin-list-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for list
      --page-size uint32    The page size (default 10)
      --page-token string   The page token. If more results are available, a "next_page" key is present in the --format=json output
      --reverse             Reverse the results
```

### Flags inherited from parent commands {#buf-beta-registry-plugin-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-plugin-list-parent-command}

* [buf beta registry plugin](#buf-beta-registry-plugin)	 - Manage Protobuf plugins

# buf beta registry plugin undeprecate {#buf-beta-registry-plugin-undeprecate}
Undeprecate a plugin

### Usage {#buf-beta-registry-plugin-undeprecate-usage} 
```terminal
$ buf beta registry plugin undeprecate <buf.build/owner/plugins/plugin> [flags]
```

### Flags {#buf-beta-registry-plugin-undeprecate-flags}

```
  -h, --help   help for undeprecate
```

### Flags inherited from parent commands {#buf-beta-registry-plugin-undeprecate-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-plugin-undeprecate-parent-command}

* [buf beta registry plugin](#buf-beta-registry-plugin)	 - Manage Protobuf plugins

# buf beta registry plugin version {#buf-beta-registry-plugin-version}
Manage Protobuf plugin versions

### Usage {#buf-beta-registry-plugin-version-usage} 
```terminal
$ buf beta registry plugin version [flags]
```

### Flags {#buf-beta-registry-plugin-version-flags}

```
  -h, --help   help for version
```

### Flags inherited from parent commands {#buf-beta-registry-plugin-version-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-plugin-version-subcommands}

* [buf beta registry plugin version list](#buf-beta-registry-plugin-version-list)	 - List plugin versions

### Parent Command {#buf-beta-registry-plugin-version-parent-command}

* [buf beta registry plugin](#buf-beta-registry-plugin)	 - Manage Protobuf plugins

# buf beta registry plugin version list {#buf-beta-registry-plugin-version-list}
List plugin versions

### Usage {#buf-beta-registry-plugin-version-list-usage} 
```terminal
$ buf beta registry plugin version list <buf.build/owner/plugins/plugin> [flags]
```

### Flags {#buf-beta-registry-plugin-version-list-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for list
      --page-size uint32    The page size (default 10)
      --page-token string   The page token. If more results are available, a "next_page" key is present in the --format=json output
      --reverse             Reverse the results
```

### Flags inherited from parent commands {#buf-beta-registry-plugin-version-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-plugin-version-list-parent-command}

* [buf beta registry plugin version](#buf-beta-registry-plugin-version)	 - Manage Protobuf plugin versions

# buf beta registry repository {#buf-beta-registry-repository}
Manage repositories

### Usage {#buf-beta-registry-repository-usage} 
```terminal
$ buf beta registry repository [flags]
```

### Flags {#buf-beta-registry-repository-flags}

```
  -h, --help   help for repository
```

### Flags inherited from parent commands {#buf-beta-registry-repository-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-repository-subcommands}

* [buf beta registry repository create](#buf-beta-registry-repository-create)	 - Create a BSR repository
* [buf beta registry repository delete](#buf-beta-registry-repository-delete)	 - Delete a BSR repository
* [buf beta registry repository deprecate](#buf-beta-registry-repository-deprecate)	 - Deprecate a BSR repository
* [buf beta registry repository get](#buf-beta-registry-repository-get)	 - Get a BSR repository
* [buf beta registry repository list](#buf-beta-registry-repository-list)	 - List BSR repositories
* [buf beta registry repository undeprecate](#buf-beta-registry-repository-undeprecate)	 - Undeprecate a BSR repository
* [buf beta registry repository update](#buf-beta-registry-repository-update)	 - Update BSR repository settings

### Parent Command {#buf-beta-registry-repository-parent-command}

* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry

# buf beta registry repository create {#buf-beta-registry-repository-create}
Create a BSR repository

### Usage {#buf-beta-registry-repository-create-usage} 
```terminal
$ buf beta registry repository create <buf.build/owner/repository> [flags]
```

### Flags {#buf-beta-registry-repository-create-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for create
      --visibility string   The repository's visibility setting. Must be one of [public,private].
```

### Flags inherited from parent commands {#buf-beta-registry-repository-create-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-repository-create-parent-command}

* [buf beta registry repository](#buf-beta-registry-repository)	 - Manage repositories

# buf beta registry repository delete {#buf-beta-registry-repository-delete}
Delete a BSR repository

### Usage {#buf-beta-registry-repository-delete-usage} 
```terminal
$ buf beta registry repository delete <buf.build/owner/repository> [flags]
```

### Flags {#buf-beta-registry-repository-delete-flags}

```
      --force   Force deletion without confirming. Use with caution
  -h, --help    help for delete
```

### Flags inherited from parent commands {#buf-beta-registry-repository-delete-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-repository-delete-parent-command}

* [buf beta registry repository](#buf-beta-registry-repository)	 - Manage repositories

# buf beta registry repository deprecate {#buf-beta-registry-repository-deprecate}
Deprecate a BSR repository

### Usage {#buf-beta-registry-repository-deprecate-usage} 
```terminal
$ buf beta registry repository deprecate <buf.build/owner/repository> [flags]
```

### Flags {#buf-beta-registry-repository-deprecate-flags}

```
  -h, --help             help for deprecate
      --message string   The message to display with deprecation warnings
```

### Flags inherited from parent commands {#buf-beta-registry-repository-deprecate-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-repository-deprecate-parent-command}

* [buf beta registry repository](#buf-beta-registry-repository)	 - Manage repositories

# buf beta registry repository get {#buf-beta-registry-repository-get}
Get a BSR repository

### Usage {#buf-beta-registry-repository-get-usage} 
```terminal
$ buf beta registry repository get <buf.build/owner/repository> [flags]
```

### Flags {#buf-beta-registry-repository-get-flags}

```
      --format string   The output format to use. Must be one of [text,json] (default "text")
  -h, --help            help for get
```

### Flags inherited from parent commands {#buf-beta-registry-repository-get-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-repository-get-parent-command}

* [buf beta registry repository](#buf-beta-registry-repository)	 - Manage repositories

# buf beta registry repository list {#buf-beta-registry-repository-list}
List BSR repositories

### Usage {#buf-beta-registry-repository-list-usage} 
```terminal
$ buf beta registry repository list <buf.build> [flags]
```

### Flags {#buf-beta-registry-repository-list-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for list
      --page-size uint32    The page size. (default 10)
      --page-token string   The page token. If more results are available, a "next_page" key is present in the --format=json output
      --reverse             Reverse the results
```

### Flags inherited from parent commands {#buf-beta-registry-repository-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-repository-list-parent-command}

* [buf beta registry repository](#buf-beta-registry-repository)	 - Manage repositories

# buf beta registry repository undeprecate {#buf-beta-registry-repository-undeprecate}
Undeprecate a BSR repository

### Usage {#buf-beta-registry-repository-undeprecate-usage} 
```terminal
$ buf beta registry repository undeprecate <buf.build/owner/repository> [flags]
```

### Flags {#buf-beta-registry-repository-undeprecate-flags}

```
  -h, --help   help for undeprecate
```

### Flags inherited from parent commands {#buf-beta-registry-repository-undeprecate-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-repository-undeprecate-parent-command}

* [buf beta registry repository](#buf-beta-registry-repository)	 - Manage repositories

# buf beta registry repository update {#buf-beta-registry-repository-update}
Update BSR repository settings

### Usage {#buf-beta-registry-repository-update-usage} 
```terminal
$ buf beta registry repository update <buf.build/owner/repository> [flags]
```

### Flags {#buf-beta-registry-repository-update-flags}

```
  -h, --help                help for update
      --visibility string   The repository's visibility setting. Must be one of [public,private].
```

### Flags inherited from parent commands {#buf-beta-registry-repository-update-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-repository-update-parent-command}

* [buf beta registry repository](#buf-beta-registry-repository)	 - Manage repositories

# buf beta registry tag {#buf-beta-registry-tag}
Manage a repository's tags

### Usage {#buf-beta-registry-tag-usage} 
```terminal
$ buf beta registry tag [flags]
```

### Flags {#buf-beta-registry-tag-flags}

```
  -h, --help   help for tag
```

### Flags inherited from parent commands {#buf-beta-registry-tag-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-tag-subcommands}

* [buf beta registry tag create](#buf-beta-registry-tag-create)	 - Create a tag for a specified commit
* [buf beta registry tag list](#buf-beta-registry-tag-list)	 - List repository tags

### Parent Command {#buf-beta-registry-tag-parent-command}

* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry

# buf beta registry tag create {#buf-beta-registry-tag-create}
Create a tag for a specified commit

### Usage {#buf-beta-registry-tag-create-usage} 
```terminal
$ buf beta registry tag create <buf.build/owner/repository:commit> <tag> [flags]
```

### Flags {#buf-beta-registry-tag-create-flags}

```
      --format string   The output format to use. Must be one of [text,json] (default "text")
  -h, --help            help for create
```

### Flags inherited from parent commands {#buf-beta-registry-tag-create-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-tag-create-parent-command}

* [buf beta registry tag](#buf-beta-registry-tag)	 - Manage a repository's tags

# buf beta registry tag list {#buf-beta-registry-tag-list}
List repository tags

### Usage {#buf-beta-registry-tag-list-usage} 
```terminal
$ buf beta registry tag list <buf.build/owner/repository> [flags]
```

### Flags {#buf-beta-registry-tag-list-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for list
      --page-size uint32    The page size. (default 10)
      --page-token string   The page token. If more results are available, a "next_page" key is present in the --format=json output
      --reverse             Reverse the results
```

### Flags inherited from parent commands {#buf-beta-registry-tag-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-tag-list-parent-command}

* [buf beta registry tag](#buf-beta-registry-tag)	 - Manage a repository's tags

# buf beta registry template {#buf-beta-registry-template}
Manage Protobuf templates on the Buf Schema Registry

### Usage {#buf-beta-registry-template-usage} 
```terminal
$ buf beta registry template [flags]
```

### Flags {#buf-beta-registry-template-flags}

```
  -h, --help   help for template
```

### Flags inherited from parent commands {#buf-beta-registry-template-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-template-subcommands}

* [buf beta registry template create](#buf-beta-registry-template-create)	 - Create a Buf template
* [buf beta registry template delete](#buf-beta-registry-template-delete)	 - Delete a template
* [buf beta registry template deprecate](#buf-beta-registry-template-deprecate)	 - Deprecate a template
* [buf beta registry template list](#buf-beta-registry-template-list)	 - List templates on the specified BSR
* [buf beta registry template undeprecate](#buf-beta-registry-template-undeprecate)	 - Undeprecate a template
* [buf beta registry template version](#buf-beta-registry-template-version)	 - Manage Protobuf template versions

### Parent Command {#buf-beta-registry-template-parent-command}

* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry

# buf beta registry template create {#buf-beta-registry-template-create}
Create a Buf template

### Usage {#buf-beta-registry-template-create-usage} 
```terminal
$ buf beta registry template create <buf.build/owner/templates/template> [flags]
```

### Flags {#buf-beta-registry-template-create-flags}

```
      --config string       The template file or data to use for configuration. Must be in either YAML or JSON format
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for create
      --visibility string   The template's visibility setting. Must be one of [public,private]
```

### Flags inherited from parent commands {#buf-beta-registry-template-create-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-template-create-parent-command}

* [buf beta registry template](#buf-beta-registry-template)	 - Manage Protobuf templates on the Buf Schema Registry

# buf beta registry template delete {#buf-beta-registry-template-delete}
Delete a template

### Usage {#buf-beta-registry-template-delete-usage} 
```terminal
$ buf beta registry template delete <buf.build/owner/templates/template> [flags]
```

### Flags {#buf-beta-registry-template-delete-flags}

```
      --force   Force deletion without confirming. Use with caution
  -h, --help    help for delete
```

### Flags inherited from parent commands {#buf-beta-registry-template-delete-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-template-delete-parent-command}

* [buf beta registry template](#buf-beta-registry-template)	 - Manage Protobuf templates on the Buf Schema Registry

# buf beta registry template deprecate {#buf-beta-registry-template-deprecate}
Deprecate a template

### Usage {#buf-beta-registry-template-deprecate-usage} 
```terminal
$ buf beta registry template deprecate <buf.build/owner/templates/template> [flags]
```

### Flags {#buf-beta-registry-template-deprecate-flags}

```
  -h, --help             help for deprecate
      --message string   The message to display with deprecation warnings
```

### Flags inherited from parent commands {#buf-beta-registry-template-deprecate-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-template-deprecate-parent-command}

* [buf beta registry template](#buf-beta-registry-template)	 - Manage Protobuf templates on the Buf Schema Registry

# buf beta registry template list {#buf-beta-registry-template-list}
List templates on the specified BSR

### Usage {#buf-beta-registry-template-list-usage} 
```terminal
$ buf beta registry template list <buf.build> [flags]
```

### Flags {#buf-beta-registry-template-list-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for list
      --page-size uint32    The page size. (default 10)
      --page-token string   The page token. If more results are available, a "next_page" key is present in the --format=json output
      --reverse             Reverse the results.
```

### Flags inherited from parent commands {#buf-beta-registry-template-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-template-list-parent-command}

* [buf beta registry template](#buf-beta-registry-template)	 - Manage Protobuf templates on the Buf Schema Registry

# buf beta registry template undeprecate {#buf-beta-registry-template-undeprecate}
Undeprecate a template

### Usage {#buf-beta-registry-template-undeprecate-usage} 
```terminal
$ buf beta registry template undeprecate <buf.build/owner/templates/template> [flags]
```

### Flags {#buf-beta-registry-template-undeprecate-flags}

```
  -h, --help   help for undeprecate
```

### Flags inherited from parent commands {#buf-beta-registry-template-undeprecate-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-template-undeprecate-parent-command}

* [buf beta registry template](#buf-beta-registry-template)	 - Manage Protobuf templates on the Buf Schema Registry

# buf beta registry template version {#buf-beta-registry-template-version}
Manage Protobuf template versions

### Usage {#buf-beta-registry-template-version-usage} 
```terminal
$ buf beta registry template version [flags]
```

### Flags {#buf-beta-registry-template-version-flags}

```
  -h, --help   help for version
```

### Flags inherited from parent commands {#buf-beta-registry-template-version-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-template-version-subcommands}

* [buf beta registry template version create](#buf-beta-registry-template-version-create)	 - Create a new template version
* [buf beta registry template version list](#buf-beta-registry-template-version-list)	 - List versions for the specified template

### Parent Command {#buf-beta-registry-template-version-parent-command}

* [buf beta registry template](#buf-beta-registry-template)	 - Manage Protobuf templates on the Buf Schema Registry

# buf beta registry template version create {#buf-beta-registry-template-version-create}
Create a new template version

### Usage {#buf-beta-registry-template-version-create-usage} 
```terminal
$ buf beta registry template version create <buf.build/owner/templates/template> [flags]
```

### Flags {#buf-beta-registry-template-version-create-flags}

```
      --config string   The template file or data to use for configuration. Must be in either YAML or JSON format
      --format string   The output format to use. Must be one of [text,json] (default "text")
  -h, --help            help for create
      --name string     The name of the new template version
```

### Flags inherited from parent commands {#buf-beta-registry-template-version-create-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-template-version-create-parent-command}

* [buf beta registry template version](#buf-beta-registry-template-version)	 - Manage Protobuf template versions

# buf beta registry template version list {#buf-beta-registry-template-version-list}
List versions for the specified template

### Usage {#buf-beta-registry-template-version-list-usage} 
```terminal
$ buf beta registry template version list <buf.build/owner/templates/template> [flags]
```

### Flags {#buf-beta-registry-template-version-list-flags}

```
      --format string       The output format to use. Must be one of [text,json] (default "text")
  -h, --help                help for list
      --page-size uint32    The page size. (default 10)
      --page-token string   The page token. If more results are available, a "next_page" key is present in the --format=json output
      --reverse             Reverse the results.
```

### Flags inherited from parent commands {#buf-beta-registry-template-version-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-template-version-list-parent-command}

* [buf beta registry template version](#buf-beta-registry-template-version)	 - Manage Protobuf template versions

# buf beta registry webhook {#buf-beta-registry-webhook}
Manage webhooks for a repository on the Buf Schema Registry

### Usage {#buf-beta-registry-webhook-usage} 
```terminal
$ buf beta registry webhook [flags]
```

### Flags {#buf-beta-registry-webhook-flags}

```
  -h, --help   help for webhook
```

### Flags inherited from parent commands {#buf-beta-registry-webhook-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-beta-registry-webhook-subcommands}

* [buf beta registry webhook create](#buf-beta-registry-webhook-create)	 - Create a repository webhook
* [buf beta registry webhook delete](#buf-beta-registry-webhook-delete)	 - Delete a repository webhook
* [buf beta registry webhook list](#buf-beta-registry-webhook-list)	 - List repository webhooks

### Parent Command {#buf-beta-registry-webhook-parent-command}

* [buf beta registry](#buf-beta-registry)	 - Manage assets on the Buf Schema Registry

# buf beta registry webhook create {#buf-beta-registry-webhook-create}
Create a repository webhook

### Usage {#buf-beta-registry-webhook-create-usage} 
```terminal
$ buf beta registry webhook create [flags]
```

### Flags {#buf-beta-registry-webhook-create-flags}

```
      --callback-url string   The url for the webhook to callback to on a given event
      --event string          The event type to create a webhook for. The proto enum string value is used for this input (e.g. 'WEBHOOK_EVENT_REPOSITORY_PUSH')
  -h, --help                  help for create
      --owner string          The owner name of the repository to create a webhook for
      --remote string         The remote of the repository the created webhook will belong to
      --repository string     The repository name to create a webhook for
```

### Flags inherited from parent commands {#buf-beta-registry-webhook-create-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-webhook-create-parent-command}

* [buf beta registry webhook](#buf-beta-registry-webhook)	 - Manage webhooks for a repository on the Buf Schema Registry

# buf beta registry webhook delete {#buf-beta-registry-webhook-delete}
Delete a repository webhook

### Usage {#buf-beta-registry-webhook-delete-usage} 
```terminal
$ buf beta registry webhook delete [flags]
```

### Flags {#buf-beta-registry-webhook-delete-flags}

```
  -h, --help            help for delete
      --id string       The webhook ID to delete
      --remote string   The remote of the repository the webhook ID belongs to
```

### Flags inherited from parent commands {#buf-beta-registry-webhook-delete-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-webhook-delete-parent-command}

* [buf beta registry webhook](#buf-beta-registry-webhook)	 - Manage webhooks for a repository on the Buf Schema Registry

# buf beta registry webhook list {#buf-beta-registry-webhook-list}
List repository webhooks

### Usage {#buf-beta-registry-webhook-list-usage} 
```terminal
$ buf beta registry webhook list [flags]
```

### Flags {#buf-beta-registry-webhook-list-flags}

```
  -h, --help                help for list
      --owner string        The owner name of the repository to list webhooks for
      --remote string       The remote of the owner and repository to list webhooks for
      --repository string   The repository name to list webhooks for.
```

### Flags inherited from parent commands {#buf-beta-registry-webhook-list-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-registry-webhook-list-parent-command}

* [buf beta registry webhook](#buf-beta-registry-webhook)	 - Manage webhooks for a repository on the Buf Schema Registry

# buf beta studio-agent {#buf-beta-studio-agent}
Run an HTTP(S) server as the Studio agent

### Usage {#buf-beta-studio-agent-usage} 
```terminal
$ buf beta studio-agent [flags]
```

### Flags {#buf-beta-studio-agent-flags}

```
      --bind string                     The address to be exposed to accept HTTP requests (default "127.0.0.1")
      --ca-cert string                  The CA cert to be used in the client and server TLS configuration
      --client-cert string              The cert to be used in the client TLS configuration
      --client-key string               The key to be used in the client TLS configuration
      --disallowed-header strings       The header names that are disallowed by this agent. When the agent receives an enveloped request with these headers set, it will return an error rather than forward the request to the target server. Multiple headers are appended if specified multiple times
      --forward-header stringToString   The headers to be forwarded via the agent to the target server. Must be an equals sign separated key-value pair (like --forward-header=fromHeader1=toHeader1). Multiple header pairs are appended if specified multiple times (default [])
  -h, --help                            help for studio-agent
      --origin string                   The allowed origin for CORS options (default "https://studio.buf.build")
      --port string                     The port to be exposed to accept HTTP requests (default "8080")
      --private-network                 Use the agent with private network CORS
      --server-cert string              The cert to be used in the server TLS configuration
      --server-key string               The key to be used in the server TLS configuration
```

### Flags inherited from parent commands {#buf-beta-studio-agent-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-beta-studio-agent-parent-command}

* [buf beta](#buf-beta)	 - Beta commands. Unstable and likely to change

# buf breaking {#buf-breaking}
Verify no breaking changes have been made

### Usage {#buf-breaking-usage} 
```terminal
$ buf breaking <input> --against <against-input> [flags]
```

### Description {#buf-breaking-description}

buf breaking makes sure that the &lt;input&gt; location has no breaking changes compared to the &lt;against-input&gt; location. The first argument is the source, module, or image to check for breaking changes.
The first argument must be one of format [bin,dir,git,json,mod,protofile,tar,zip].
Defaults to &#34;.&#34; if no argument is specified.
 

### Flags {#buf-breaking-flags}

```
      --against string          Required. The source, module, or image to check against. Must be one of format [bin,dir,git,json,mod,protofile,tar,zip]
      --against-config string   The file or data to use to configure the against source, module, or image
      --config string           The file or data to use for configuration
      --disable-symlinks        Do not follow symlinks when reading sources or configuration from the local filesystem
                                By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry
      --error-format string     The format for build errors or check violations printed to stdout. Must be one of [text,json,msvs,junit] (default "text")
      --exclude-imports         Exclude imports from breaking change detection.
      --exclude-path strings    Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                                If specified multiple times, the union is taken
  -h, --help                    help for breaking
      --limit-to-input-files    Only run breaking checks against the files in the input
                                When set, the against input contains only the files in the input
                                Overrides --path
      --path strings            Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                                If specified multiple times, the union is taken
```

### Flags inherited from parent commands {#buf-breaking-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-breaking-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf build {#buf-build}
Build Protobuf files into a Buf image

### Usage {#buf-build-usage} 
```terminal
$ buf build <input> [flags]
```

### Description {#buf-build-description}

The first argument is the source or module to build or image to convert.
The first argument must be one of format [bin,dir,git,json,mod,protofile,tar,zip].
Defaults to &#34;.&#34; if no argument is specified.
 

### Flags {#buf-build-flags}

```
      --as-file-descriptor-set   Output as a google.protobuf.FileDescriptorSet instead of an image
                                 Note that images are wire compatible with FileDescriptorSets, but this flag strips
                                 the additional metadata added for Buf usage
      --config string            The file or data to use to use for configuration
      --disable-symlinks         Do not follow symlinks when reading sources or configuration from the local filesystem
                                 By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry
      --error-format string      The format for build errors printed to stderr. Must be one of [text,json,msvs,junit] (default "text")
      --exclude-imports          Exclude imports.
      --exclude-path strings     Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                                 If specified multiple times, the union is taken
      --exclude-source-info      Exclude source info
  -h, --help                     help for build
  -o, --output string            The output location for the built image. Must be one of format [bin,json] (default "/dev/null")
      --path strings             Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                                 If specified multiple times, the union is taken
      --type strings             The types (message, enum, service) that should be included in this image. When specified, the resulting image will only include descriptors to describe the requested types
```

### Flags inherited from parent commands {#buf-build-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-build-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf convert {#buf-convert}
Convert a message from binary to JSON or vice versa

### Usage {#buf-convert-usage} 
```terminal
$ buf convert <input> [flags]
```

### Description {#buf-convert-description}

Use an input proto to interpret a proto/json message and convert it to a different format.

Examples:

```terminal
$ buf convert <input> --type=<type> --from=<payload> --to=<output>
```

The &lt;input&gt; can be a local .proto file, binary output of &#34;buf build&#34;, bsr module or local buf module:

```terminal
$ buf convert example.proto --type=Foo.proto --from=payload.json --to=output.bin
```

All of &lt;input&gt;, &#34;--from&#34; and &#34;to&#34; accept formatting options:

```terminal
$ buf convert example.proto#format=bin --type=buf.Foo --from=payload#format=json --to=out#format=json
```

Both &lt;input&gt; and &#34;--from&#34; accept stdin redirecting:

```terminal
$ buf convert <(buf build -o -)#format=bin --type=foo.Bar --from=<(echo "{\"one\":\"55\"}")#format=json
```

Redirect from stdin to --from:

```terminal
$ echo "{\"one\":\"55\"}" | buf convert buf.proto --type buf.Foo --from -#format=json
```

Redirect from stdin to &lt;input&gt;:

```terminal
$ buf build -o - | buf convert -#format=bin --type buf.Foo --from=payload.json
```

Use a module on the bsr:

```terminal
$ buf convert <buf.build/owner/repository> --type buf.Foo --from=payload.json
```
 

### Flags {#buf-convert-flags}

```
      --error-format string   The format for build errors printed to stderr. Must be one of [text,json,msvs,junit] (default "text")
      --from string           The location of the payload to be converted. Supported formats are [bin,json] (default "-")
  -h, --help                  help for convert
      --to string             The output location of the conversion. Supported formats are [bin,json] (default "-")
      --type string           The full type name of the message within the input (e.g. acme.weather.v1.Units)
```

### Flags inherited from parent commands {#buf-convert-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-convert-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf curl {#buf-curl}
Invoke an RPC endpoint, a la 'cURL'

### Usage {#buf-curl-usage} 
```terminal
$ buf curl <url> [flags]
```

### Description {#buf-curl-description}

This command helps you invoke HTTP RPC endpoints on a server that uses gRPC or Connect.

By default, server reflection is used, unless the --reflect flag is set to false. Without server
reflection, a --schema flag must be provided to indicate the Protobuf schema for the method being
invoked.

The only positional argument is the URL of the RPC method to invoke. The name of the method to
invoke comes from the last two path components of the URL, which should be the fully-qualified
service name and method name, respectively.

The URL can use either http or https as the scheme. If http is used then HTTP 1.1 will be used
unless the --http2-prior-knowledge flag is set. If https is used then HTTP/2 will be preferred
during protocol negotiation and HTTP 1.1 used only if the server does not support HTTP/2.

The default RPC protocol used will be Connect. To use a different protocol (gRPC or gRPC-Web),
use the --protocol flag. Note that the gRPC protocol cannot be used with HTTP 1.1.

The input request is specified via the -d or --data flag. If absent, an empty request is sent. If
the flag value starts with an at-sign (@), then the rest of the flag value is interpreted as a
filename from which to read the request body. If that filename is just a dash (-), then the request
body is read from stdin. The request body is a JSON document that contains the JSON formatted
request message. If the RPC method being invoked is a client-streaming method, the request body may
consist of multiple JSON values, appended to one another. Multiple JSON documents should usually be
separated by whitespace, though this is not strictly required unless the request message type has a
custom JSON representation that is not a JSON object.

Request metadata (i.e. headers) are defined using -H or --header flags. The flag value is in
&#34;name: value&#34; format. But if it starts with an at-sign (@), the rest of the value is interpreted as
a filename from which headers are read, each on a separate line. If the filename is just a dash (-),
then the headers are read from stdin.

If headers and the request body are both to be read from the same file (or both read from stdin),
the file must include headers first, then a blank line, and then the request body.

Examples:

Issue a unary RPC to a plain-text (i.e. &#34;h2c&#34;) gRPC server, where the schema for the service is
in a Buf module in the current directory, using an empty request message:

```terminal
$ buf curl --schema . --protocol grpc --http2-prior-knowledge  \
     http://localhost:20202/foo.bar.v1.FooService/DoSomething
```

Issue an RPC to a Connect server, where the schema comes from the Buf Schema Registry, using
a request that is defined as a command-line argument:

```terminal
$ buf curl --schema buf.build/bufbuild/eliza  \
     --data '{"name": "Bob Loblaw"}'          \
     https://demo.connect.build/buf.connect.demo.eliza.v1.ElizaService/Introduce
```

Issue a unary RPC to a server that supports reflection, with verbose output:

```terminal
$ buf curl --data '{"sentence": "I am not feeling well."}' -v  \
     https://demo.connect.build/buf.connect.demo.eliza.v1.ElizaService/Say
```

Issue a client-streaming RPC to a gRPC-web server that supports reflection, where custom
headers and request data are both in a heredoc:

```terminal
$ buf curl --data @- --header @- --protocol grpcweb                              \
     https://demo.connect.build/buf.connect.demo.eliza.v1.ElizaService/Converse  \
   <<EOM
Custom-Header-1: foo-bar-baz
Authorization: token jas8374hgnkvje9wpkerebncjqol4

{"sentence": "Hi, doc. I feel hungry."}
{"sentence": "What is the answer to life, the universe, and everything?"}
{"sentence": "If you were a fish, what of fish would you be?."}
EOM
```

Note that server reflection (i.e. use of the --reflect flag) does not work with HTTP 1.1 since the
protocol relies on bidirectional streaming. If server reflection is used, the assumed URL for the
reflection service is the same as the given URL, but with the last two elements removed and
replaced with the service and method name for server reflection.

If an error occurs that is due to incorrect usage or other unexpected error, this program will
return an exit code that is less than 8. If the RPC fails otherwise, this program will return an
exit code that is the gRPC code, shifted three bits to the left.
 

### Flags {#buf-curl-flags}

```
      --cacert string             Path to a PEM-encoded X509 certificate pool file that contains the set of trusted
                                  certificate authorities/issuers. If omitted, the system's default set of trusted
                                  certificates are used to verify the server's certificate. This option is only valid
                                  when the URL uses the https scheme. It is not applicable if --insecure flag is used
  -E, --cert string               Path to a PEM-encoded X509 certificate file, for using client certificates with TLS. This
                                  option is only valid when the URL uses the https scheme. A --key flag must also be
                                  present to provide tha private key that corresponds to the given certificate
      --connect-timeout float     The time limit, in seconds, for a connection to be established with the server. There is
                                  no limit if this flag is not present
  -d, --data string               Request data. This should be zero or more JSON documents, each indicating a request
                                  message. For unary RPCs, there should be exactly one JSON document. A special value of
                                  '@<path>' means to read the data from the file at <path>. If the path is "-" then the
                                  request data is read from stdin. If the same file is indicated as used with the request
                                  headers flags (--header or -H), the file must contain all headers, then a blank line, and
                                  then the request body. It is not allowed to indicate stdin if the schema is expected to be
                                  provided via stdin as a file descriptor set or image
  -H, --header strings            Request headers to include with the RPC invocation. This flag may be specified more
                                  than once to indicate multiple headers. Each flag value should have the form "name: value".
                                  A special value of '@<path>' means to read headers from the file at <path>. If the path
                                  is "-" then headers are read from stdin. If the same file is indicated as used with the
                                  request data flag (--data or -d), the file must contain all headers, then a blank line,
                                  and then the request body. It is not allowed to indicate stdin if the schema is expected
                                  to be provided via stdin as a file descriptor set or image
  -h, --help                      help for curl
      --http2-prior-knowledge     This flag can be used with URLs that use the http scheme (as opposed to https) to indicate
                                  that HTTP/2 should be used. Without this, HTTP 1.1 will be used with URLs with an http
                                  scheme. For https scheme, HTTP/2 will be negotiate during the TLS handshake if the server
                                  supports it (otherwise HTTP 1.1 is used)
  -k, --insecure                  If set, the TLS connection will be insecure and the server's certificate will NOT be
                                  verified. This is generally discouraged. This option is only valid when the URL uses
                                  the https scheme
      --keepalive-time float      The duration, in seconds, between TCP keepalive transmissions (default 60)
      --key string                Path to a PEM-encoded X509 private key file, for using client certificates with TLS. This
                                  option is only valid when the URL uses the https scheme. A --cert flag must also be
                                  present to provide tha certificate and public key that corresponds to the given
                                  private key
      --no-keepalive              By default, connections are created using TCP keepalive. If this flag is present, they
                                  will be disabled
  -o, --output string             Path to output file to create with response data. If absent, response is printed to stdout
      --protocol string           The RPC protocol to use. This can be one of "grpc", "grpcweb", or "connect" (default "connect")
      --reflect                   If true, use server reflection to determine the schema (default true)
      --reflect-header strings    Request headers to include with reflection requests. This flag may only be used
                                  when --reflect is also set. This flag may be specified more than once to indicate
                                  multiple headers. Each flag value should have the form "name: value". But a special value
                                  of '*' may be used to indicate that all normal request headers (from --header and -H
                                  flags) should also be included with reflection requests. A special value of '@<path>'
                                  means to read headers from the file at <path>. If the path is "-" then headers are
                                  read from stdin. It is not allowed to indicate a file with the same path as used with
                                  the request data flag (--data or -d). Furthermore, it is not allowed to indicate stdin
                                  if the schema is expected to be provided via stdin as a file descriptor set or image
      --reflect-protocol string   The reflection protocol to use for downloading information from the server. This flag
                                  may only be used when server reflection is used. By default, this command will try all known
                                  reflection protocols from newest to oldest. If this results in a "Not Implemented" error,
                                  then older protocols will be used. In practice, this means that "grpc-v1" is tried first,
                                  and "grpc-v1alpha" is used if it doesn't work. If newer reflection protocols are introduced,
                                  they may be preferred in the absence of this flag being explicitly set to a specific protocol.
                                  The valid values for this flag are "grpc-v1" and "grpc-v1alpha". These correspond to services
                                  named "grpc.reflection.v1.ServerReflection" and "grpc.reflection.v1alpha.ServerReflection"
                                  respectively
      --schema string             The module to use for the RPC schema. This is necessary if the server does not support
                                  server reflection. The format of this argument is the same as for the <input> arguments to
                                  other buf sub-commands such as build and generate. It can indicate a directory, a file, a
                                  remote module in the Buf Schema Registry, or even standard in ("-") for feeding an image or
                                  file descriptor set to the command in a shell pipeline.
                                  Setting this flags implies --reflect=false
      --servername string         The server name to use in TLS handshakes (for SNI) if the URL scheme is https. If not
                                  specified, the default is the origin host in the URL or the value in a "Host" header if
                                  one is provided
      --unix-socket string        The path to a unix socket that will be used instead of opening a TCP socket to the host
                                  and port indicated in the URL
  -A, --user-agent string         The user agent string to send
```

### Flags inherited from parent commands {#buf-curl-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-curl-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf export {#buf-export}
Export proto files from one location to another

### Usage {#buf-export-usage} 
```terminal
$ buf export <source> [flags]
```

### Description {#buf-export-description}

The first argument is the source or module to export.
The first argument must be one of format [dir,git,mod,protofile,tar,zip].
Defaults to &#34;.&#34; if no argument is specified.

Examples:

Export proto files in &lt;source&gt; to an output directory.

```terminal
$ buf export <source> --output=<output-dir>
```

Export current directory to another local directory. 

```terminal
$ buf export . --output=<output-dir>
```

Export the latest remote module to a local directory.

```terminal
$ buf export <buf.build/owner/repository> --output=<output-dir>
```

Export a specific version of a remote module to a local directory.

```terminal
$ buf export <buf.build/owner/repository:ref> --output=<output-dir>
```

Export a git repo to a local directory.

```terminal
$ buf export https://github.com/owner/repository.git --output=<output-dir>
```
 

### Flags {#buf-export-flags}

```
      --config string          The file or data to use for configuration
      --disable-symlinks       Do not follow symlinks when reading sources or configuration from the local filesystem
                               By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry
      --exclude-imports        Exclude imports.
      --exclude-path strings   Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
  -h, --help                   help for export
  -o, --output string          The output directory for exported files
      --path strings           Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
```

### Flags inherited from parent commands {#buf-export-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-export-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf format {#buf-format}
Format Protobuf files

### Usage {#buf-format-usage} 
```terminal
$ buf format <source> [flags]
```

### Description {#buf-format-description}

By default, the source is the current directory and the formatted content is written to stdout.

Examples:

Write the current directory&#39;s formatted content to stdout:

```terminal
$ buf format
```

Most people will want to rewrite the files defined in the current directory in-place with -w:

```terminal
$ buf format -w
```

Display a diff between the original and formatted content with -d
Write a diff instead of the formatted file:
```terminal

$ buf format simple/simple.proto -d

$ diff -u simple/simple.proto.orig simple/simple.proto
--- simple/simple.proto.orig	2022-03-24 09:44:10.000000000 -0700
+++ simple/simple.proto	2022-03-24 09:44:10.000000000 -0700
@@ -2,8 +2,7 @@

 package simple;

-
 message Object {
-    string key = 1;
-   bytes value = 2;
+  string key = 1;
+  bytes value = 2;
 }
```

Use the --exit-code flag to exit with a non-zero exit code if there is a diff:

```terminal
$ buf format --exit-code
$ buf format -w --exit-code
$ buf format -d --exit-code
```

Format a file, directory, or module reference by specifying a source e.g.
Write the formatted file to stdout:
```terminal

$ buf format simple/simple.proto

syntax = "proto3";

package simple;

message Object {
  string key = 1;
  bytes value = 2;
}
```

Write the formatted directory to stdout:

```terminal
$ buf format simple
...
```

Write the formatted module reference to stdout:

```terminal
$ buf format buf.build/acme/petapis
...
```

Write the result to a specified output file or directory with -o e.g.

Write the formatted file to another file:

```terminal
$ buf format simple/simple.proto -o simple/simple.formatted.proto
```

Write the formatted directory to another directory, creating it if it doesn&#39;t exist:

```terminal
$ buf format proto -o formatted
```

This also works with module references:

```terminal
$ buf format buf.build/acme/weather -o formatted
```

Rewrite the file(s) in-place with -w. e.g.

Rewrite a single file in-place:

```terminal
$ buf format simple.proto -w
```

Rewrite an entire directory in-place:

```terminal
$ buf format proto -w
```

Write a diff and rewrite the file(s) in-place:

```terminal
$ buf format simple -d -w

$ diff -u simple/simple.proto.orig simple/simple.proto
...
```

The -w and -o flags cannot be used together in a single invocation.
 

### Flags {#buf-format-flags}

```
      --config string          The file or data to use for configuration
  -d, --diff                   Display diffs instead of rewriting files
      --disable-symlinks       Do not follow symlinks when reading sources or configuration from the local filesystem
                               By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry
      --error-format string    The format for build errors printed to stderr. Must be one of [text,json,msvs,junit] (default "text")
      --exclude-path strings   Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
      --exit-code              Exit with a non-zero exit code if files were not already formatted
  -h, --help                   help for format
  -o, --output string          The output location for the formatted files. Must be one of format [dir,git,protofile,tar,zip]. If omitted, the result is written to stdout (default "-")
      --path strings           Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
  -w, --write                  Rewrite files in-place
```

### Flags inherited from parent commands {#buf-format-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-format-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf generate {#buf-generate}
Generate code with protoc plugins

### Usage {#buf-generate-usage} 
```terminal
$ buf generate <input> [flags]
```

### Description {#buf-generate-description}

This command uses a template file of the shape:

```terminal
# buf.gen.yaml
# The version of the generation template.
# Required.
# The valid values are v1beta1, v1.
version: v1
# The plugins to run. "plugin" is required.
plugins:
    # The name of the plugin.
    # By default, buf generate will look for a binary named protoc-gen-NAME on your $PATH.
    # Alternatively, use a remote plugin:
    # plugin: buf.build/protocolbuffers/go:v1.28.1
  - plugin: go
    # The the relative output directory.
    # Required.
    out: gen/go
    # Any options to provide to the plugin.
    # This can be either a single string or a list of strings.
    # Optional.
    opt: paths=source_relative
    # The custom path to the plugin binary, if not protoc-gen-NAME on your $PATH.
    # Optional, and exclusive with "remote".
    path: custom-gen-go
    # The generation strategy to use. There are two options:
    #
    # 1. "directory"
    #
    #   This will result in buf splitting the input files by directory, and making separate plugin
    #   invocations in parallel. This is roughly the concurrent equivalent of:
    #
    #     for dir in $(find . -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq); do
    #       protoc -I . $(find "${dir}" -name '*.proto')
    #     done
    #
    #   Almost every Protobuf plugin either requires this, or works with this,
    #   and this is the recommended and default value.
    #
    # 2. "all"
    #
    #   This will result in buf making a single plugin invocation with all input files.
    #   This is roughly the equivalent of:
    #
    #     protoc -I . $(find . -name '*.proto')
    #
    #   This is needed for certain plugins that expect all files to be given at once.
    #
    # If omitted, "directory" is used. Most users should not need to set this option.
    # Optional.
    strategy: directory
  - plugin: java
    out: gen/java
    # Use the plugin hosted at buf.build/protocolbuffers/python at version v21.9.
    # If version is omitted, uses the latest version of the plugin.
  - plugin: buf.build/protocolbuffers/python:v21.9
    out: gen/python
```

As an example, here&#39;s a typical &#34;buf.gen.yaml&#34; go and grpc, assuming
&#34;protoc-gen-go&#34; and &#34;protoc-gen-go-grpc&#34; are on your &#34;$PATH&#34;:

```terminal
# buf.gen.yaml
version: v1
plugins:
  - plugin: go
    out: gen/go
    opt: paths=source_relative
  - plugin: go-grpc
    out: gen/go
    opt: paths=source_relative,require_unimplemented_servers=false
```

By default, buf generate will look for a file of this shape named
&#34;buf.gen.yaml&#34; in your current directory. This can be thought of as a template
for the set of plugins you want to invoke.

The first argument is the source, module, or image to generate from.
Defaults to &#34;.&#34; if no argument is specified.

Use buf.gen.yaml as template, current directory as input:

```terminal
$ buf generate
```

Same as the defaults (template of &#34;buf.gen.yaml&#34;, current directory as input):

```terminal
$ buf generate --template buf.gen.yaml .
```

The --template flag also takes YAML or JSON data as input, so it can be used without a file:

```terminal
$ buf generate --template '{"version":"v1","plugins":[{"plugin":"go","out":"gen/go"}]}'
```

Download the repository and generate code stubs per the bar.yaml template:

```terminal
$ buf generate --template bar.yaml https://github.com/foo/bar.git
```

Generate to the bar/ directory, prepending bar/ to the out directives in the template:

```terminal
$ buf generate --template bar.yaml -o bar https://github.com/foo/bar.git
```

The paths in the template and the -o flag will be interpreted as relative to the
current directory, so you can place your template files anywhere.

If you only want to generate stubs for a subset of your input, you can do so via the --path. e.g.

Only generate for the files in the directories proto/foo and proto/bar:

```terminal
$ buf generate --path proto/foo --path proto/bar
```

Only generate for the files proto/foo/foo.proto and proto/foo/bar.proto:

```terminal
$ buf generate --path proto/foo/foo.proto --path proto/foo/bar.proto
```

Only generate for the files in the directory proto/foo on your git repository:

```terminal
$ buf generate --template buf.gen.yaml https://github.com/foo/bar.git --path proto/foo
```

Note that all paths must be contained within the same module. For example, if you have a
module in &#34;proto&#34;, you cannot specify &#34;--path proto&#34;, however &#34;--path proto/foo&#34; is allowed
as &#34;proto/foo&#34; is contained within &#34;proto&#34;.

Plugins are invoked in the order they are specified in the template, but each plugin
has a per-directory parallel invocation, with results from each invocation combined
before writing the result.

Insertion points are processed in the order the plugins are specified in the template.
 

### Flags {#buf-generate-flags}

```
      --config string          The file or data to use for configuration
      --disable-symlinks       Do not follow symlinks when reading sources or configuration from the local filesystem
                               By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry
      --error-format string    The format for build errors, printed to stderr. Must be one of [text,json,msvs,junit] (default "text")
      --exclude-path strings   Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
  -h, --help                   help for generate
      --include-imports        Also generate all imports except for Well-Known Types
      --include-wkt            Also generate Well-Known Types. Cannot be set without --include-imports
  -o, --output string          The base directory to generate to. This is prepended to the out directories in the generation template (default ".")
      --path strings           Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
      --template string        The generation template file or data to use. Must be in either YAML or JSON format
      --type strings           The types (message, enum, service) that should be included in this image. When specified, the resulting image will only include descriptors to describe the requested types. Flag usage overrides buf.gen.yaml
```

### Flags inherited from parent commands {#buf-generate-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-generate-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf lint {#buf-lint}
Run linting on Protobuf files

### Usage {#buf-lint-usage} 
```terminal
$ buf lint <input> [flags]
```

### Description {#buf-lint-description}

The first argument is the source, module, or Image to lint.
The first argument must be one of format [bin,dir,git,json,mod,protofile,tar,zip].
Defaults to &#34;.&#34; if no argument is specified.
 

### Flags {#buf-lint-flags}

```
      --config string          The file or data to use for configuration
      --disable-symlinks       Do not follow symlinks when reading sources or configuration from the local filesystem
                               By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry
      --error-format string    The format for build errors or check violations printed to stdout. Must be one of [text,json,msvs,junit,config-ignore-yaml] (default "text")
      --exclude-path strings   Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
  -h, --help                   help for lint
      --path strings           Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
```

### Flags inherited from parent commands {#buf-lint-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-lint-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf mod {#buf-mod}
Manage Buf modules

### Usage {#buf-mod-usage} 
```terminal
$ buf mod [flags]
```

### Flags {#buf-mod-flags}

```
  -h, --help   help for mod
```

### Flags inherited from parent commands {#buf-mod-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-mod-subcommands}

* [buf mod clear-cache](#buf-mod-clear-cache)	 - Clear Buf module cache
* [buf mod init](#buf-mod-init)	 - Initializes and writes a new buf.yaml configuration file.
* [buf mod ls-breaking-rules](#buf-mod-ls-breaking-rules)	 - List breaking rules
* [buf mod ls-lint-rules](#buf-mod-ls-lint-rules)	 - List lint rules
* [buf mod open](#buf-mod-open)	 - Open the module's homepage in a web browser
* [buf mod prune](#buf-mod-prune)	 - Prune unused dependencies from the buf.lock file
* [buf mod update](#buf-mod-update)	 - Update a module's dependencies by updating the buf.lock file

### Parent Command {#buf-mod-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf mod clear-cache {#buf-mod-clear-cache}
Clear Buf module cache

### Usage {#buf-mod-clear-cache-usage} 
```terminal
$ buf mod clear-cache [flags]
```

### Flags {#buf-mod-clear-cache-flags}

```
  -h, --help   help for clear-cache
```

### Flags inherited from parent commands {#buf-mod-clear-cache-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-mod-clear-cache-parent-command}

* [buf mod](#buf-mod)	 - Manage Buf modules

# buf mod init {#buf-mod-init}
Initializes and writes a new buf.yaml configuration file.

### Usage {#buf-mod-init-usage} 
```terminal
$ buf mod init [buf.build/owner/foobar] [flags]
```

### Flags {#buf-mod-init-flags}

```
      --doc             Write inline documentation in the form of comments in the resulting configuration file
  -h, --help            help for init
  -o, --output string   The directory to write the configuration file to (default ".")
```

### Flags inherited from parent commands {#buf-mod-init-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-mod-init-parent-command}

* [buf mod](#buf-mod)	 - Manage Buf modules

# buf mod ls-breaking-rules {#buf-mod-ls-breaking-rules}
List breaking rules

### Usage {#buf-mod-ls-breaking-rules-usage} 
```terminal
$ buf mod ls-breaking-rules [flags]
```

### Flags {#buf-mod-ls-breaking-rules-flags}

```
      --all              List all rules and not just those currently configured
      --config string    The file or data to use for configuration. Ignored if --all or --version is specified
      --format string    The format to print rules as. Must be one of [text,json] (default "text")
  -h, --help             help for ls-breaking-rules
      --version string   List all the rules for the given configuration version. Implies --all. Must be one of [v1beta1,v1]
```

### Flags inherited from parent commands {#buf-mod-ls-breaking-rules-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-mod-ls-breaking-rules-parent-command}

* [buf mod](#buf-mod)	 - Manage Buf modules

# buf mod ls-lint-rules {#buf-mod-ls-lint-rules}
List lint rules

### Usage {#buf-mod-ls-lint-rules-usage} 
```terminal
$ buf mod ls-lint-rules [flags]
```

### Flags {#buf-mod-ls-lint-rules-flags}

```
      --all              List all rules and not just those currently configured
      --config string    The file or data to use for configuration. Ignored if --all or --version is specified
      --format string    The format to print rules as. Must be one of [text,json] (default "text")
  -h, --help             help for ls-lint-rules
      --version string   List all the rules for the given configuration version. Implies --all. Must be one of [v1beta1,v1]
```

### Flags inherited from parent commands {#buf-mod-ls-lint-rules-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-mod-ls-lint-rules-parent-command}

* [buf mod](#buf-mod)	 - Manage Buf modules

# buf mod open {#buf-mod-open}
Open the module's homepage in a web browser

### Usage {#buf-mod-open-usage} 
```terminal
$ buf mod open <directory> [flags]
```

### Description {#buf-mod-open-description}

The first argument is the directory of the local module to open. Defaults to &#34;.&#34; if no argument is specified.
 

### Flags {#buf-mod-open-flags}

```
  -h, --help   help for open
```

### Flags inherited from parent commands {#buf-mod-open-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-mod-open-parent-command}

* [buf mod](#buf-mod)	 - Manage Buf modules

# buf mod prune {#buf-mod-prune}
Prune unused dependencies from the buf.lock file

### Usage {#buf-mod-prune-usage} 
```terminal
$ buf mod prune <directory> [flags]
```

### Description {#buf-mod-prune-description}

The first argument is the directory of the local module to prune. Defaults to &#34;.&#34; if no argument is specified.
 

### Flags {#buf-mod-prune-flags}

```
  -h, --help   help for prune
```

### Flags inherited from parent commands {#buf-mod-prune-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-mod-prune-parent-command}

* [buf mod](#buf-mod)	 - Manage Buf modules

# buf mod update {#buf-mod-update}
Update a module's dependencies by updating the buf.lock file

### Usage {#buf-mod-update-usage} 
```terminal
$ buf mod update <directory> [flags]
```

### Description {#buf-mod-update-description}

Fetch the latest digests for the specified references in the config file, and write them and their transitive dependencies to the buf.lock file. The first argument is the directory of the local module to update. Defaults to &#34;.&#34; if no argument is specified.
 

### Flags {#buf-mod-update-flags}

```
  -h, --help           help for update
      --only strings   The name of the dependency to update. When set, only this dependency is updated (along with any of its sub-dependencies). May be passed multiple times
```

### Flags inherited from parent commands {#buf-mod-update-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-mod-update-parent-command}

* [buf mod](#buf-mod)	 - Manage Buf modules

# buf push {#buf-push}
Push a module to a registry

### Usage {#buf-push-usage} 
```terminal
$ buf push <source> [flags]
```

### Description {#buf-push-description}

The first argument is the source to push.
The first argument must be one of format [dir,git,protofile,tar,zip].
Defaults to &#34;.&#34; if no argument is specified.
 

### Flags {#buf-push-flags}

```
      --disable-symlinks      Do not follow symlinks when reading sources or configuration from the local filesystem
                              By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry
      --draft string          Make the pushed commit a draft with the specified name. Cannot be used together with --tag (-t)
      --error-format string   The format for build errors printed to stderr. Must be one of [text,json,msvs,junit] (default "text")
  -h, --help                  help for push
  -t, --tag strings           Create a tag for the pushed commit. Multiple tags are created if specified multiple times. Cannot be used together with --draft
```

### Flags inherited from parent commands {#buf-push-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-push-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf registry {#buf-registry}
Manage assets on the Buf Schema Registry

### Usage {#buf-registry-usage} 
```terminal
$ buf registry [flags]
```

### Flags {#buf-registry-flags}

```
  -h, --help   help for registry
```

### Flags inherited from parent commands {#buf-registry-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Subcommands {#buf-registry-subcommands}

* [buf registry login](#buf-registry-login)	 - Log in to the Buf Schema Registry
* [buf registry logout](#buf-registry-logout)	 - Log out of the Buf Schema Registry

### Parent Command {#buf-registry-parent-command}

* [buf](#buf)	 - The Buf CLI

# buf registry login {#buf-registry-login}
Log in to the Buf Schema Registry

### Usage {#buf-registry-login-usage} 
```terminal
$ buf registry login <domain> [flags]
```

### Description {#buf-registry-login-description}

This prompts for your BSR username and a BSR token and updates your .netrc file with these credentials.
The &lt;domain&gt; argument will default to buf.build if not specified.
 

### Flags {#buf-registry-login-flags}

```
  -h, --help              help for login
      --token-stdin       Read the token from stdin. This command prompts for a token by default
      --username string   The username to use. This command prompts for a username by default
```

### Flags inherited from parent commands {#buf-registry-login-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-registry-login-parent-command}

* [buf registry](#buf-registry)	 - Manage assets on the Buf Schema Registry

# buf registry logout {#buf-registry-logout}
Log out of the Buf Schema Registry

### Usage {#buf-registry-logout-usage} 
```terminal
$ buf registry logout [flags]
```

### Description {#buf-registry-logout-description}

This command removes any BSR credentials from your .netrc file
 

### Flags {#buf-registry-logout-flags}

```
  -h, --help   help for logout
```

### Flags inherited from parent commands {#buf-registry-logout-persistent-flags}

```
      --debug               Turn on debug logging
      --log-format string   The log format [text,color,json] (default "color")
      --timeout duration    The duration until timing out (default 2m0s)
  -v, --verbose             Turn on verbose mode
```

### Parent Command {#buf-registry-logout-parent-command}

* [buf registry](#buf-registry)	 - Manage assets on the Buf Schema Registry
