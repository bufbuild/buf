package bufmod

import (
	"context"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// What is a Module?
// It definitely includes its .proto files.
// It includes its list of resolved dependencies ie buf.lock
// It includes its LICENSE - this is part of the module, a change in the license is a change to the module
// It includes its README - documentation is either in comments or in readme, both are part of the module
// It does not include breaking or lint config - this doesn't comprise module data, this is just used to operate on the module itself in certain situations (CLI)
// It does not include excludes - these are already excluded when building the module
// It does not include generation config obviously
// Basically, if you find yourself saying "X operates on the module", it's probably not part of the module itself (ie lint config, breaking config, excludes)
//
// What is a Workspace?
// It includes all of its modules
// It includes its list of resolved dependencies?
// What about LICENSE and README? A workspace isn't licensed, a Module is. A workspace isn't documented, a Module is.
//
// What about config?
// Only the CLI cares. This isn't part of Module or Workspace at all.
//
// What about --path/--exclude-path?
// Only the CLI cares.
//
// What about declared dependencies?
// Only the CLI cares.
//
// What about excludes?
// Only the CLI cares.
//
// What does this lead to?
// buf.yaml should not be part of the digest.
// Our Workspace and Module types should fall as above.
// We should figure out how to move Config to something like CommandMeta below, similar to ModuleConfig/ImageConfig/ModuleConfigSet. This should be a type outside of bufwire.
// We should figure out a nice way to deal with TargetFileInfos at a level outside of Module and Workspace.

type CommandMeta interface {
	Workspace() Workspace
	// For buf.yaml v2, ModuleIdentity is ignored
	DeclaredDepModuleReferences() []ModuleReference
	// Need some way to not do this
	TargetPaths(moduleID string) []string
	Config() *bufconfig.Config
}

type ModuleName interface{}
type ModuleReference interface{}
type ModulePin interface{}

type Workspace interface {
	Modules() []Module
	Deps() []ModulePin
}

type FileInfo interface {
	storage.ObjectInfo

	IsImport() bool
	Module() Module
}

type File interface {
	FileInfo
	io.ReadCloser
}

type Module interface {
	TargetFileInfos(context.Context) (FileInfo, error)
	SourceFileInfos(context.Context) (FileInfo, error)
	GetFile(ctx context.Context, path string) (File, error)
	Documentation() string
	DocumentationPath() string
	License() string
	Workspace() Workspace

	ID() string
	Name() ModuleName
	Commit() string
}
