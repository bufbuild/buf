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
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L75
	messageTypeTagInFile int32 = 4
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L78
	extensionTagInFile = 7
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L97
	fieldTagInMessage = 2
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L100
	nestedTypeTagInMessage = 3
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L98
	extensionTagInMessage = 6
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L215
	optionsTagInField = 8
)

func isPathForFieldOptions(path []int32) bool {
	isStateAccept := false
	currentState := start
	for _, index := range path {
		if currentState == nil {
			return false
		}
		currentState, isStateAccept = currentState(index)
	}
	return isStateAccept
}

type dfaState func(int32) (next dfaState, isNextAccept bool)

func start(index int32) (dfaState, bool) {
	switch index {
	case messageTypeTagInFile:
		return messages, false
	case extensionTagInFile:
		return fields, false
	default:
		return nil, false
	}
}

func messages(index int32) (dfaState, bool) {
	// we are not checking index >= 0, the caller must ensure this
	return message, false
}

func message(index int32) (dfaState, bool) {
	switch index {
	case nestedTypeTagInMessage:
		return messages, false
	case fieldTagInMessage, extensionTagInMessage:
		return fields, false
	}
	return nil, false
}

func fields(index int32) (dfaState, bool) {
	// we are not checking index >= 0, the caller must ensure this
	return field, false
}

func field(index int32) (dfaState, bool) {
	switch index {
	case optionsTagInField:
		return nil, true
	default:
		return nil, false
	}
}
