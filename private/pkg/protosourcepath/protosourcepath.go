// Copyright 2020-2024 Buf Technologies, Inc.
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

package protosourcepath

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	fileDescriptorProtoMessageTypeTag = int32(4)
	descriptorProtoNameTag            = int32(1)
)

var (
	errInvalidSourcePath = errors.New("invalid sourcepa")
)

// GetAssociatedSourcePaths gets the associated source paths.
func GetAssociatedSourcePaths(sourcePath protoreflect.SourcePath) ([]protoreflect.SourcePath, error) {
	if len(sourcePath) == 0 {
		return nil, nil
	}
	switch sourcePath[0] {
	case fileDescriptorProtoMessageTypeTag:
		switch len(sourcePath) {
		case 1:
			return nil, newInvalidSourcePathError(sourcePath, "expected message_type index")
		case 2:
			// No additional source paths.
			// We could argue that name is an additional source path, but let's just go with parents for now.
			return []protoreflect.SourcePath{
				sourcePath,
			}, nil
		case 3:
			switch sourcePath[2] {
			case descriptorProtoNameTag:
				return []protoreflect.SourcePath{
					sourcePath,
					// Also include message
					sourcePath[0:2],
				}, nil
			}
		}
	}
	return nil, newInvalidSourcePathError(sourcePath, "unrecognized")
}

func newInvalidSourcePathError(sourcePath protoreflect.SourcePath, format string, args ...any) error {
	return fmt.Errorf("invalid protoreflect.SourcePath %v: %s", sourcePath, fmt.Sprintf(format, args...))
}
