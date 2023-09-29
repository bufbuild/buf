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

package buflintvalidate

import (
	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/reflect/protodesc"
)

func CheckCelInFile(
	resolver protodesc.Resolver,
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	file protosource.File,
) error {
	for _, message := range file.Messages() {
		if err := checkCelInMessage(resolver, add, message); err != nil {
			return err
		}
	}
	for _, extensionField := range file.Extensions() {
		if err := checkCelInField(resolver, add, extensionField); err != nil {
			return err
		}
	}
	return nil
}
