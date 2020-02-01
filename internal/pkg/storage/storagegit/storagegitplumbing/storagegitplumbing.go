package storagegitplumbing

import "gopkg.in/src-d/go-git.v4/plumbing"

// RefName is a git reference name.
type RefName interface {
	// ReferenceName returns the go-git ReferenceName.
	ReferenceName() plumbing.ReferenceName
	String() string
}

// NewBranchRefName returns a new branch RefName.
func NewBranchRefName(branch string) RefName {
	return newRefName(plumbing.NewBranchReferenceName(branch))
}

// NewTagRefName returns a new tag RefName.
func NewTagRefName(tag string) RefName {
	return newRefName(plumbing.NewTagReferenceName(tag))
}

type refName struct {
	referenceName plumbing.ReferenceName
}

func newRefName(referenceName plumbing.ReferenceName) *refName {
	return &refName{
		referenceName: referenceName,
	}
}

func (r *refName) ReferenceName() plumbing.ReferenceName {
	return r.referenceName
}

func (r *refName) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

func (r *refName) String() string {
	if r == nil {
		return ""
	}
	return r.referenceName.String()
}
