// Copyright 2020 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

var (
	_ ParsedArchiveRef = &archiveRef{}
)

type archiveRef struct {
	format          string
	path            string
	fileScheme      FileScheme
	archiveType     ArchiveType
	compressionType CompressionType
	stripComponents uint32
}

func newArchiveRef(
	format string,
	path string,
	archiveType ArchiveType,
	compressionType CompressionType,
	stripComponents uint32,
) (*archiveRef, error) {
	if archiveType == ArchiveTypeZip && compressionType != CompressionTypeNone {
		return nil, NewCannotSpecifyCompressionForZipError()
	}
	singleRef, err := newSingleRef(
		format,
		path,
		compressionType,
	)
	if err != nil {
		return nil, err
	}
	return newDirectArchiveRef(
		singleRef.Format(),
		singleRef.Path(),
		singleRef.FileScheme(),
		archiveType,
		singleRef.CompressionType(),
		stripComponents,
	), nil
}

func newDirectArchiveRef(
	format string,
	path string,
	fileScheme FileScheme,
	archiveType ArchiveType,
	compressionType CompressionType,
	stripComponents uint32,
) *archiveRef {
	return &archiveRef{
		format:          format,
		path:            path,
		fileScheme:      fileScheme,
		archiveType:     archiveType,
		compressionType: compressionType,
		stripComponents: stripComponents,
	}
}

func (r *archiveRef) Format() string {
	return r.format
}

func (r *archiveRef) Path() string {
	return r.path
}

func (r *archiveRef) FileScheme() FileScheme {
	return r.fileScheme
}

func (r *archiveRef) ArchiveType() ArchiveType {
	return r.archiveType
}

func (r *archiveRef) CompressionType() CompressionType {
	return r.compressionType
}

func (r *archiveRef) StripComponents() uint32 {
	return r.stripComponents
}

func (*archiveRef) ref()        {}
func (*archiveRef) fileRef()    {}
func (*archiveRef) bucketRef()  {}
func (*archiveRef) archiveRef() {}
