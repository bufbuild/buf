package bufmodule

import (
	"errors"
	"fmt"
)

// ModuleRef is an unresolved reference to a Module.
//
// It can refer to the latest released commit, a different commit, a branch, a tag, or a VCS commit.
type ModuleRef interface {
	// String returns "registry/owner/name[:ref]".
	fmt.Stringer

	// ModuleFullName returns the full name of the Module.
	//
	// Always present.
	ModuleFullName() ModuleFullName
	// Ref returns the reference within the Module.
	//
	//   If Ref is empty, this refers to the latest released Commit on the Module.
	//   If Ref is a commit ID, this refers to this commit.
	//   If Ref is a tag ID or name, this refers to the commit associated with the tag.
	//   If Ref is a VCS commit ID or hash, this refers to the commit associated with the VCS commit.
	//   If Ref is a digest, this referenced to the latested released Commit with this digest.
	//   If Ref is a branch ID or name, this refers to the latest commit on the branch.
	//     If there is a conflict between names across resources (for example, there is a
	//     branch and tag with the same name), the following order of precedence is applied:
	//       - commit
	//       - VCS commit
	//       - tag
	//       - branch
	//
	// May be empty, as documented above.
	Ref() string

	isModuleRef()
}

// *** PRIVATE ***

type moduleRef struct {
	moduleFullName ModuleFullName
	ref            string
}

func newModuleRef(
	moduleFullName ModuleFullName,
	ref string,
) (*moduleRef, error) {
	if moduleFullName == nil {
		return nil, errors.New("new ModuleRef: ModuleFullName is nil")
	}
	return &moduleRef{
		moduleFullName: moduleFullName,
		ref:            ref,
	}, nil
}

func (m *moduleRef) ModuleFullName() ModuleFullName {
	return m.moduleFullName
}

func (m *moduleRef) Ref() string {
	return m.ref
}

func (m *moduleRef) String() string {
	if m.ref == "" {
		return m.moduleFullName.String()
	}
	return m.moduleFullName.String() + ":" + m.ref
}

func (*moduleRef) isModuleRef() {}
