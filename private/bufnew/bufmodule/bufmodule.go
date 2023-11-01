package bufmodule

import (
	"context"
	"errors"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/storage"
)

var (
	// ErrNotExist is the error returned if a File retrieved does not exist.
	//
	// Ese errors.Is(err,
	ErrNotExist = errors.New("file does not exist")
)

type ModuleFullName interface {
	Remote() string
	Owner() string
	Name() string

	isModuleFullName()
}

type ModuleRef interface {
	ModuleFullName() ModuleFullName
	Ref() string

	isModuleReference()
}

type ModulePin interface {
	ModuleFullName() ModuleFullName
	CommitID() string
	Digest() bufcas.Digest

	isModulePin()
}

type ModuleSet interface {
	GetModule(moduleID string) (Module, error)
	Modules() []Module
	Deps() []ModulePin
	// Needed for v1 vs v2
	IsDepsOnModule() bool

	isModuleSet()
}

type FileInfo interface {
	storage.ObjectInfo

	Module() Module

	isFileInfo()
}

type File interface {
	FileInfo
	io.ReadCloser

	isFile()
}

type Module interface {
	// GetFile gets the File within the Module as specified by the path.
	//
	// Returns an error that satisfies storage.IsNotExi
	GetFile(ctx context.Context, path string) (File, error)
	StatFile(ctx context.Context, path string) (FileInfo, error)
	WalkFiles(ctx context.Context, f func(FileInfo) error) error

	ModuleSet() ModuleSet
	// Returns all deps in ModuleSet on v2
	Deps() []ModulePin

	Ref() string
	FullName() ModuleFullName
	CommitID() string

	isModule()
}
