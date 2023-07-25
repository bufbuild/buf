// Copyright 2020-2023 Buf Technologies, Inc.
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

const (
	pathTypeInvalid pathType = iota + 1
	pathTypeEmpty
	pathTypeMessages
	pathTypeMessage
	pathTypeFields
	pathTypeField
	pathTypeFieldOptions
	pathTypeFieldOption
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L75
	messageTypeTagInFile int32 = 4
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L78
	extensionTagInFile int32 = 7
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L97
	fieldTagInMessage int32 = 2
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L100
	nestedTypeTagInMessage int32 = 3
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L98
	extensionTagInMessage int32 = 6
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L215
	optionsTagInField int32 = 8
)

// pathType is the type of a message pointed to by a location path.
type pathType int

// locationPathDFAState takes an input and returns the next state and the path type
// that ends with the input, which is consistent with the returned next state. It
// returns nil and pathTypeInvalid if the input is rejected.
type locationPathDFAState func(int32) (locationPathDFAState, pathType)

// getPathType returns the type of the path. It only accepts one of:
// empty, messages, message, fields, field, field options and field option.
func getPathType(path []int32) pathType {
	pathType := pathTypeEmpty
	currentState := start
	for _, element := range path {
		if currentState == nil {
			return pathTypeInvalid
		}
		currentState, pathType = currentState(element)
	}
	return pathType
}

func start(input int32) (locationPathDFAState, pathType) {
	switch input {
	case messageTypeTagInFile:
		return messages, pathTypeMessages
	case extensionTagInFile:
		return fields, pathTypeFields
	default:
		return nil, pathTypeInvalid
	}
}

func messages(input int32) (locationPathDFAState, pathType) {
	// we are not checking index >= 0, the caller must ensure this
	return message, pathTypeMessages
}

func message(input int32) (locationPathDFAState, pathType) {
	switch input {
	case nestedTypeTagInMessage:
		return messages, pathTypeMessage
	case fieldTagInMessage, extensionTagInMessage:
		return fields, pathTypeMessage
	}
	return nil, pathTypeInvalid
}

func fields(input int32) (locationPathDFAState, pathType) {
	// we are not checking index >= 0, the caller must ensure this
	return field, pathTypeField
}

func field(input int32) (locationPathDFAState, pathType) {
	switch input {
	case optionsTagInField:
		return fieldOptions, pathTypeFieldOptions
	default:
		return nil, pathTypeInvalid
	}
}

func fieldOptions(input int32) (locationPathDFAState, pathType) {
	return fieldOption, pathTypeFieldOption
}

func fieldOption(input int32) (locationPathDFAState, pathType) {
	// technically it should not just accpet any input, but this is good
	// enough for our purposes as the paths marked by a MarkSweeper will
	// be valid.
	return fieldOption, pathTypeFieldOption
}
