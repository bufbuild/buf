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

package registryv1alpha1

import (
	errors "errors"
	fmt "fmt"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/validate/v1alpha1"
	cel "github.com/google/cel-go/cel"
	proto "google.golang.org/protobuf/proto"
	strconv "strconv"
)

// Validate validates this message according to the given CEL expression, where 'this' is this instance.
//
//  has(this.owner_name) && has(this.repository_name) && has(this.deprecation_message)
func (x *DeprecateRepositoryByNameRequest) Validate() error {
	return validateDeprecateRepositoryByNameRequest(x)
}

var validateDeprecateRepositoryByNameRequest func(*DeprecateRepositoryByNameRequest) error

func init() {
	defaultMessage := &DeprecateRepositoryByNameRequest{}
	options := defaultMessage.ProtoReflect().Descriptor().Options()
	expr, ok := proto.GetExtension(options, v1alpha1.E_Expr).(*v1alpha1.Expr)
	if !ok {
		panic(fmt.Errorf("expected CEL expression, but got %T", expr))
	}
	env, err := cel.NewEnv(
		cel.Variable(
			"this",
			cel.ObjectType(string(defaultMessage.ProtoReflect().Descriptor().FullName())),
		),
		cel.TypeDescs(File_buf_alpha_registry_v1alpha1_repository_proto),
	)
	if err != nil {
		panic(err)
	}
	ast, issues := env.Compile(expr.GetExpression())
	if err := issues.Err(); err != nil {
		panic(err)
	}
	program, err := env.Program(ast)
	if err != nil {
		panic(err)
	}

	validateDeprecateRepositoryByNameRequest = func(x *DeprecateRepositoryByNameRequest) error {
		val, _, err := program.Eval(
			map[string]interface{}{
				"this": x,
			},
		)
		if val == nil {
			return err
		}
		value := val.Value()
		if boolVal, ok := value.(bool); ok && boolVal {
			return nil
		}
		if stringVal, ok := value.(string); ok {
			boolVal, err := strconv.ParseBool(stringVal)
			if err != nil {
				return errors.New(stringVal)
			}
			if boolVal {
				return nil
			}
		}
		return errors.New("DeprecateRepositoryByNameRequest is invalid; see the message definition for details")
	}
}
