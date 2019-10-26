package protodesc

import "github.com/bufbuild/buf/internal/pkg/errs"

type fileImport struct {
	descriptor

	imp      string
	isPublic bool
	isWeak   bool
	path     []int32
}

func newFileImport(
	descriptor descriptor,
	imp string,
	path []int32,
) (*fileImport, error) {
	if imp == "" {
		return nil, errs.NewInternalf("no dependency value in %q", descriptor.filePath)
	}
	return &fileImport{
		descriptor: descriptor,
		imp:        imp,
		path:       path,
	}, nil
}

func (f *fileImport) Import() string {
	return f.imp
}

func (f *fileImport) IsPublic() bool {
	return f.isPublic
}

func (f *fileImport) IsWeak() bool {
	return f.isWeak
}

func (f *fileImport) Location() Location {
	return f.getLocation(f.path)
}

func (f *fileImport) setIsPublic() {
	f.isPublic = true
}

func (f *fileImport) setIsWeak() {
	f.isWeak = true
}
