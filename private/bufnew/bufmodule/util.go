package bufmodule

import "sort"

const (
	// licenseFilePath is the path of the license file within a Module.
	licenseFilePath = "LICENSE"
)

var (
	// orderedDocFilePaths are the potential documentation file paths for a Module.
	//
	// When creating a Module from a Bucket, we check the file paths buf.md, README.md, and README.markdown
	// to exist, in that order. The first one to exist is chosen as the documentation file that is considered
	// part of the Module, and any others are discarded.
	orderedDocFilePaths = []string{
		"buf.md",
		"README.md",
		"README.markdown",
	}
)

func uniqueSortedModulePins(modulePins []ModulePin) []ModulePin {
	// Note that we do not error check that the same dependency has a common commit within
	// this function - it is the responsibility of the ModuleSet/Module constructor to make
	// sure this property is true.
	modulePinKeyToValue := make(map[modulePinKey]modulePinValue, len(modulePins))
	for _, modulePin := range modulePinStringToModulePin {
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
	uniqueSortedModulePins := make([]ModulePin, len(uniqueModulePinValues))
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
