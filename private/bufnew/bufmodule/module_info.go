package bufmodule

import (
	"context"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// ModuleInfo contains identifying information for a Module.
//
// It is embedded inside a Module, and therefore is always available from FileInfos as well.
// It can also be used to get Modules with the ModuleProvider.
type ModuleInfo interface {
	// ModuleFullName returns the full name of the Module.
	//
	// May be nil depending on context. For example, when read from lock files, this will
	// never be nil, however on Modules, it may be. You should check if this is nil when
	// performing operations, and error if you have a different expectation.
	ModuleFullName() ModuleFullName
	// CommitID returns the ID of the Commit, if present.
	//
	// This is an ID of a Commit on the BSR, and can be used in API operations.
	//
	// May be empty depending on context. For example, when read from lock files, this will
	// never be empty, however on Modules, it may be. You should check if this is empty when
	// performing operations, and error if you have a different expectation.
	//
	// If ModuleFullName is nil, this will always be empty.
	CommitID() string
	// Digest returns the Module digest.
	Digest(context.Context) (bufcas.Digest, error)

	isModuleInfo()
}

func ModuleInfoToModuleRef(moduleInfo ModuleInfo) ModuleRef {
	//return newModuleRefNoValidate(
	//moduleInfo.Registry(),
	//moduleInfo.Owner(),
	//moduleInfo.Name(),
	//moduleInfo.CommitID(),
	//)
	return nil
}

func ProtoCommitToModuleInfo(protoCommit *modulev1beta1.Commit) (ModuleInfo, error) {
	return nil, nil
}
