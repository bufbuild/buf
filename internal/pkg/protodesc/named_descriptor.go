package protodesc

import (
	"fmt"
	"strings"
)

type namedDescriptor struct {
	locationDescriptor

	name        string
	namePath    []int32
	nestedNames []string
}

func newNamedDescriptor(
	locationDescriptor locationDescriptor,
	name string,
	namePath []int32,
	nestedNames []string,
) (namedDescriptor, error) {
	if name == "" {
		return namedDescriptor{}, fmt.Errorf("no name in %q", locationDescriptor.filePath)
	}
	return namedDescriptor{
		locationDescriptor: locationDescriptor,
		name:               name,
		namePath:           namePath,
		nestedNames:        nestedNames,
	}, nil
}

func (n *namedDescriptor) FullName() string {
	if n.Package() != "" {
		return n.Package() + "." + n.NestedName()
	}
	return n.NestedName()
}

func (n *namedDescriptor) NestedName() string {
	if len(n.nestedNames) == 0 {
		return n.Name()
	}
	return strings.Join(n.nestedNames, ".") + "." + n.Name()
}

func (n *namedDescriptor) Name() string {
	return n.name
}

func (n *namedDescriptor) NameLocation() Location {
	return n.getLocation(n.namePath)
}
