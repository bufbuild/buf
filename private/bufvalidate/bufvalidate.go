// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufvalidate

import (
	"context"

	"google.golang.org/protobuf/compiler/protogen"
)

// Validator is the interface implemented by messages that define
// the (buf.alpha.v1alpha1.validate.expr) option.
type Validator interface {
	Validate() error
}

// Generator generates validation code from in-line CEL annotations.
//
// https://github.com/google/cel-spec/blob/c69db37ac4a1bffd175137de6a886682e4d194d4/doc/intro.md
type Generator interface {
	Generate(ctx context.Context) error
}

// NewGenerator retruns a new Generator backed by the given plugin.
func NewGenerator(plugin *protogen.Plugin) Generator {
	return newGenerator(plugin)
}
