package protodesc

type locationDescriptor struct {
	descriptor

	path []int32
}

func newLocationDescriptor(
	descriptor descriptor,
	path []int32,
) locationDescriptor {
	return locationDescriptor{
		descriptor: descriptor,
		path:       path,
	}
}

func (l *locationDescriptor) Location() Location {
	return l.getLocation(l.path)
}
