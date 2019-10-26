package protodesc

import "github.com/bufbuild/buf/internal/pkg/errs"

type reservedName struct {
	locationDescriptor

	value string
}

func newReservedName(
	locationDescriptor locationDescriptor,
	value string,
) (*reservedName, error) {
	if value == "" {
		return nil, errs.NewInternalf("no value for reserved name in %q", locationDescriptor.filePath)
	}
	return &reservedName{
		locationDescriptor: locationDescriptor,
		value:              value,
	}, nil
}

func (r *reservedName) Value() string {
	return r.value
}
