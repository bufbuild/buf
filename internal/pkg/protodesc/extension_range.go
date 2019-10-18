package protodesc

// TODO: this is hand-computed and this is not great, we should figure out what this actually is or make a constant somewhere else
const extensionRangeMaxMinusOne = 536870911

type extensionRange struct {
	locationDescriptor

	start int // inclusive
	end   int // exclusive
}

func newExtensionRange(
	locationDescriptor locationDescriptor,
	start int,
	end int,
) *extensionRange {
	return &extensionRange{
		locationDescriptor: locationDescriptor,
		start:              start,
		end:                end,
	}
}

func (e *extensionRange) Start() int {
	return e.start
}

func (e *extensionRange) End() int {
	return e.end
}
