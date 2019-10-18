package protodesc

type enum struct {
	namedDescriptor

	values         []EnumValue
	allowAlias     bool
	allowAliasPath []int32
	reservedRanges []ReservedRange
	reservedNames  []ReservedName
}

func newEnum(
	namedDescriptor namedDescriptor,
	allowAlias bool,
	allowAliasPath []int32,
) *enum {
	return &enum{
		namedDescriptor: namedDescriptor,
		allowAlias:      allowAlias,
		allowAliasPath:  allowAliasPath,
	}
}

func (e *enum) Values() []EnumValue {
	return e.values
}

func (e *enum) AllowAlias() bool {
	return e.allowAlias
}

func (e *enum) AllowAliasLocation() Location {
	return e.getLocation(e.allowAliasPath)
}

func (e *enum) ReservedRanges() []ReservedRange {
	return e.reservedRanges
}

func (e *enum) ReservedNames() []ReservedName {
	return e.reservedNames
}

func (e *enum) addValue(value EnumValue) {
	e.values = append(e.values, value)
}

func (e *enum) addReservedRange(reservedRange ReservedRange) {
	e.reservedRanges = append(e.reservedRanges, reservedRange)
}

func (e *enum) addReservedName(reservedName ReservedName) {
	e.reservedNames = append(e.reservedNames, reservedName)
}
