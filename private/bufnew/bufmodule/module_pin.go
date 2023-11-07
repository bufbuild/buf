package bufmodule

import (
	"sort"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// ModulePin is a specific Module and commit, along with its associated Digest.
type ModulePin interface {
	// ModuleFullName returns the full name of the Module.
	ModuleFullName() ModuleFullName
	// CommitID returns the commit ID.
	//
	// This can be used as a Commit ID within the BSR.
	CommitID() string
	// Digest returns the Digest of the Module at the specific commit.
	//
	// ModuleDigestB5 is currently used to calculate Digests.
	Digest() bufcas.Digest

	isModulePin()
}

func NewModulePin(
	registry string,
	owner string,
	name string,
	commitID string,
	digest bufcas.Digest,
) (ModulePin, error) {
	return nil, nil
}

func ModulePinToModuleRef(modulePin ModulePin) ModuleRef {
	//return newModuleRefNoValidate(
	//modulePin.Registry(),
	//modulePin.Owner(),
	//modulePin.Name(),
	//modulePin.CommitID(),
	//)
	return nil
}

// this conversion should be possible.
func ModulePinToModuleInfo(modulePin ModulePin) ModuleInfo {
	return nil
}

func ProtoCommitToModulePin(protoCommit *modulev1beta1.Commit) (ModulePin, error) {
	return nil, nil
}

// *** PRIVATE ***

func uniqueSortedModulePins(modulePins []ModulePin) []ModulePin {
	// Note that we do not error check that the same dependency has a common commit within
	// this function - it is the responsibility of the ModuleSet/Module constructor to make
	// sure this property is true.
	modulePinKeyToValue := make(map[modulePinKey]modulePinValue, len(modulePins))
	for _, modulePin := range modulePins {
		modulePinValue := newModulePinValue(modulePin)
		modulePinKeyToValue[modulePinValue.modulePinKey] = modulePinValue
	}
	modulePinValues := make([]modulePinValue, 0, len(modulePinKeyToValue))
	for _, modulePinValue := range modulePinKeyToValue {
		modulePinValues = append(modulePinValues, modulePinValue)
	}
	sort.Slice(
		modulePinValues,
		func(i int, j int) bool {
			return modulePinValues[i].sortString() < modulePinValues[j].sortString()
		},
	)
	uniqueSortedModulePins := make([]ModulePin, len(modulePinValues))
	for i, modulePinValue := range modulePinValues {
		uniqueSortedModulePins[i] = modulePinValue.modulePin
	}
	return uniqueSortedModulePins
}

type modulePinKey struct {
	registry     string
	owner        string
	name         string
	commitID     string
	digestString string
}

func newModulePinKey(modulePin ModulePin) modulePinKey {
	return modulePinKey{
		registry:     modulePin.ModuleFullName().Registry(),
		owner:        modulePin.ModuleFullName().Owner(),
		name:         modulePin.ModuleFullName().Name(),
		commitID:     modulePin.CommitID(),
		digestString: modulePin.Digest().String(),
	}
}

func (k modulePinKey) sortString() string {
	return k.registry + "/" + k.owner + "/" + k.name + ":" + k.commitID + " " + k.digestString
}

type modulePinValue struct {
	modulePinKey modulePinKey
	modulePin    ModulePin
}

func newModulePinValue(modulePin ModulePin) modulePinValue {
	return modulePinValue{
		modulePinKey: newModulePinKey(modulePin),
		modulePin:    modulePin,
	}
}

func (v modulePinValue) sortString() string {
	return v.modulePinKey.sortString()
}
