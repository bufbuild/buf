package bufmodule

import "github.com/bufbuild/buf/private/bufpkg/bufcas"

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
