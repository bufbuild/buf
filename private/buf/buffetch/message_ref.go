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

package buffetch

import (
	"github.com/bufbuild/buf/private/buf/buffetch/internal"
)

var _ MessageRef = &messageRef{}

type messageRef struct {
	singleRef       internal.SingleRef
	useProtoNames   bool
	useEnumNumbers  bool
	messageEncoding MessageEncoding
}

func newMessageRef(
	singleRef internal.SingleRef,
	messageEncoding MessageEncoding,
) (*messageRef, error) {
	useProtoNames, err := getTrueOrFalseForSingleRef(singleRef, useProtoNamesKey)
	if err != nil {
		return nil, err
	}
	useEnumNumbers, err := getTrueOrFalseForSingleRef(singleRef, useEnumNumbersKey)
	if err != nil {
		return nil, err
	}
	return &messageRef{
		singleRef:       singleRef,
		useProtoNames:   useProtoNames,
		useEnumNumbers:  useEnumNumbers,
		messageEncoding: messageEncoding,
	}, nil
}

func (r *messageRef) MessageEncoding() MessageEncoding {
	return r.messageEncoding
}

func (r *messageRef) Path() string {
	return r.singleRef.Path()
}

func (r *messageRef) UseProtoNames() bool {
	return r.useProtoNames
}

func (r *messageRef) UseEnumNumbers() bool {
	return r.useEnumNumbers
}

func (r *messageRef) IsNull() bool {
	return r.singleRef.FileScheme() == internal.FileSchemeNull
}

func (r *messageRef) internalRef() internal.Ref {
	return r.singleRef
}

func (r *messageRef) internalSingleRef() internal.SingleRef {
	return r.singleRef
}

func getTrueOrFalseForSingleRef(singleRef internal.SingleRef, key string) (bool, error) {
	value, ok := singleRef.CustomOptionValue(key)
	if !ok {
		return false, nil
	}
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, internal.NewOptionsInvalidValueForKeyError(key, value)
	}
}
