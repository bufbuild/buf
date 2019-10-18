package protodesc

import protobufdescriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"

type location struct {
	sourceCodeInfoLocation *protobufdescriptor.SourceCodeInfo_Location
}

func newLocation(sourceCodeInfoLocation *protobufdescriptor.SourceCodeInfo_Location) *location {
	return &location{
		sourceCodeInfoLocation: sourceCodeInfoLocation,
	}
}

func (l *location) StartLine() int {
	switch len(l.sourceCodeInfoLocation.Span) {
	case 3, 4:
		return int(l.sourceCodeInfoLocation.Span[0]) + 1
	default:
		// since we are not erroring, making this and others 1 so that other code isn't messed up by assuming
		// this is >= 1
		return 1
	}
}

func (l *location) StartColumn() int {
	switch len(l.sourceCodeInfoLocation.Span) {
	case 3, 4:
		return int(l.sourceCodeInfoLocation.Span[1]) + 1
	default:
		// since we are not erroring, making this and others 1 so that other code isn't messed up by assuming
		// this is >= 1
		return 1
	}
}

func (l *location) EndLine() int {
	switch len(l.sourceCodeInfoLocation.Span) {
	case 3:
		return int(l.sourceCodeInfoLocation.Span[0]) + 1
	case 4:
		return int(l.sourceCodeInfoLocation.Span[2]) + 1
	default:
		// since we are not erroring, making this and others 1 so that other code isn't messed up by assuming
		// this is >= 1
		return 1
	}
}

func (l *location) EndColumn() int {
	switch len(l.sourceCodeInfoLocation.Span) {
	case 3:
		return int(l.sourceCodeInfoLocation.Span[2]) + 1
	case 4:
		return int(l.sourceCodeInfoLocation.Span[3]) + 1
	default:
		// since we are not erroring, making this and others 1 so that other code isn't messed up by assuming
		// this is >= 1
		return 1
	}
}

func (l *location) LeadingComments() string {
	if l.sourceCodeInfoLocation.LeadingComments == nil {
		return ""
	}
	return *l.sourceCodeInfoLocation.LeadingComments
}

func (l *location) TrailingComments() string {
	if l.sourceCodeInfoLocation.TrailingComments == nil {
		return ""
	}
	return *l.sourceCodeInfoLocation.TrailingComments
}

func (l *location) LeadingDetachedComments() []string {
	return l.sourceCodeInfoLocation.LeadingDetachedComments
}
