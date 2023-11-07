package bufmodule

import (
	"context"
	"sort"

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

// *** PRIVATE ***

func uniqueSortedModuleInfos(moduleInfos []ModuleInfo) []ModuleInfo {
	// Note that we do not error check that the same dependency has a common commit within
	// this function - it is the responsibility of the ModuleSet/Module constructor to make
	// sure this property is true.
	moduleInfoKeyToValue := make(map[moduleInfoKey]moduleInfoValue, len(moduleInfos))
	for _, moduleInfo := range moduleInfos {
		moduleInfoValue := newModuleInfoValue(moduleInfo)
		moduleInfoKeyToValue[moduleInfoValue.moduleInfoKey] = moduleInfoValue
	}
	moduleInfoValues := make([]moduleInfoValue, 0, len(moduleInfoKeyToValue))
	for _, moduleInfoValue := range moduleInfoKeyToValue {
		moduleInfoValues = append(moduleInfoValues, moduleInfoValue)
	}
	sort.Slice(
		moduleInfoValues,
		func(i int, j int) bool {
			return moduleInfoValues[i].sortString() < moduleInfoValues[j].sortString()
		},
	)
	uniqueSortedModuleInfos := make([]ModuleInfo, len(moduleInfoValues))
	for i, moduleInfoValue := range moduleInfoValues {
		uniqueSortedModuleInfos[i] = moduleInfoValue.moduleInfo
	}
	return uniqueSortedModuleInfos
}

type moduleInfoKey struct {
	registry     string
	owner        string
	name         string
	commitID     string
	digestString string
}

func newModuleInfoKey(moduleInfo ModuleInfo) moduleInfoKey {
	return moduleInfoKey{
		registry:     moduleInfo.ModuleFullName().Registry(),
		owner:        moduleInfo.ModuleFullName().Owner(),
		name:         moduleInfo.ModuleFullName().Name(),
		commitID:     moduleInfo.CommitID(),
		digestString: moduleInfo.Digest().String(),
	}
}

func (k moduleInfoKey) sortString() string {
	return k.registry + "/" + k.owner + "/" + k.name + ":" + k.commitID + " " + k.digestString
}

type moduleInfoValue struct {
	moduleInfoKey moduleInfoKey
	moduleInfo    ModuleInfo
}

func newModuleInfoValue(moduleInfo ModuleInfo) moduleInfoValue {
	return moduleInfoValue{
		moduleInfoKey: newModuleInfoKey(moduleInfo),
		moduleInfo:    moduleInfo,
	}
}

func (v moduleInfoValue) sortString() string {
	return v.moduleInfoKey.sortString()
}
