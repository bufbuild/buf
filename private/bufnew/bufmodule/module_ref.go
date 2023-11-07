package bufmodule

import "fmt"

// ModuleRef is an unresolved reference to a Module.
//
// It can refer to the latest released commit, a different commit, a branch, a tag, or a VCS commit.
type ModuleRef interface {
	// String returns "registry/owner/name[:ref]".
	fmt.Stringer

	// ModuleFullName returns the full name of the Module.
	ModuleFullName() ModuleFullName
	// Ref returns the reference within the Module.
	//
	//   If Ref is empty, this refers to the latest released Commit on the Module.
	//   If Ref is a commit ID, this refers to this commit.
	//   If Ref is a tag ID or name, this refers to the commit associated with the tag.
	//   If Ref is a VCS commit ID or hash, this refers to the commit associated with the VCS commit.
	//   If Ref is a branch ID or name, this refers to the latest commit on the branch.
	//     If there is a conflict between names across resources (for example, there is a
	//     branch and tag with the same name), the following order of precedence is applied:
	//       - commit
	//       - VCS commit
	//       - tag
	//       - branch
	Ref() string

	isModuleRef()
}

// *** PRIVATE ***

type moduleRef struct {
}
