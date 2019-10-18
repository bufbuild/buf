package protodesc

// TODO: this is hand-computed and this is not great, we should figure out what this actually is or make a constant somewhere else
const (
	reservedRangeInclusiveMax = 2147483647
	reservedRangeExclusiveMax = 536870911
)

type reservedRange struct {
	locationDescriptor

	start int
	end   int
	// true for messages, false for enums
	endIsExclusive bool
}

func newReservedRange(
	locationDescriptor locationDescriptor,
	start int,
	end int,
	endIsExclusive bool,
) *reservedRange {
	return &reservedRange{
		locationDescriptor: locationDescriptor,
		start:              start,
		end:                end,
		endIsExclusive:     endIsExclusive,
	}
}

func (r *reservedRange) Start() int {
	return r.start
}

func (r *reservedRange) End() int {
	return r.end
}

func (r *reservedRange) EndIsExclusive() bool {
	return r.endIsExclusive
}
