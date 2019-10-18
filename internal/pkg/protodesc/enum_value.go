package protodesc

type enumValue struct {
	namedDescriptor

	enum       Enum
	number     int
	numberPath []int32
}

func newEnumValue(
	namedDescriptor namedDescriptor,
	enum Enum,
	number int,
	numberPath []int32,
) *enumValue {
	return &enumValue{
		namedDescriptor: namedDescriptor,
		enum:            enum,
		number:          number,
		numberPath:      numberPath,
	}
}

func (e *enumValue) Enum() Enum {
	return e.enum
}

func (e *enumValue) Number() int {
	return e.number
}

func (e *enumValue) NumberLocation() Location {
	return e.getLocation(e.numberPath)
}
