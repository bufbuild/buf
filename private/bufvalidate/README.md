# protoc-gen-validate-go

This is a proof-of-concept for an alternative to [`protoc-gen-validate`](https://github.com/envoyproxy/protoc-gen-validate)
that is powered by the Common Expression Language, or [CEL](https://github.com/google/cel-spec).

With CEL, you can define constraints on the Protobuf message as a whole, not just
individual fields (which is the case for `protoc-gen-validate`). Plus, CEL is _fast_.
From the [Cel-Go codelab](https://codelabs.developers.google.com/codelabs/cel-go),
you'll notice the following statement:

> [CEL] evaluates safely on the order of nanoseconds to microseconds; it's ideal
> for performance-critical applications.

Furthermore, `protoc-gen-validate-go` provides an easy way to customize the error
message returned by your application (unlike `protoc-gen-validate`). Simply return
a `string` from your CEL expression and it will be used as the error message returned
from the `bufvalidate.Validator` interface.

## Usage

A natural use case for this tool is server-side validation for request structures
in a backend application. The following sections will break down a use case from
the perspective of the `.proto` sources themselves, as well as the generated Go code.

### Protobuf option

Simply import the `buf.alpha.validate.v1alpha1.expr` option (shown below), and define
a CEL expression that enforces one or more constraints:

```protobuf
// buf/alpha/validate/v1alpha1/validate.proto
syntax = "proto2";

package buf.alpha.validate.v1alpha1;

import "google/protobuf/descriptor.proto";

extend google.protobuf.MessageOptions {
  // expr is a CEL expression.
  //
  // The expression is validated against the
  // set of fields defined in the message.
  optional Expr expr = 8556;
}

// Expr is a a textual expression in the Common Expression Language (CEL) syntax.
// This type is inspired by and compatible with the type hosted in GoogleAPIs.
//
// Ref: https://github.com/googleapis/googleapis/blob/072a9467b63dc46db4a24cfaad1c2f33e3c508d2/google/type/expr.proto#L56
message Expr {
  optional string expression = 1;
  optional string title = 2;
  optional string description = 3;
  optional string location = 4;
}
```

```protobuf
// example/v1/object.proto
syntax = "proto3";

pacakge example.v1;

import "buf/alpha/validate/v1alpha1/validate.proto";

message Object {
  string key = 1;
  bytes value = 2;
}

message CreateObjectRequest {
  option (buf.alpha.validate.v1alpha1.expr) = {
    description: "Validates that all of the required fields are set",
    expression:
      "has(this.key) && "
      "has(this.value)"
  }
  string key = 1;
  bytes value = 2;
}
```

Note the only required field is the `expression` - the `description` is optional.

### Generated Go code

Now that the `buf.alpha.validate.v1alpha1.expr` option is set, the `protoc-gen-validate-go`
plugin will generate a file that implements the `bufvalidate.Validator` interface:

```yaml
# buf.gen.yaml
version: v1
managed:
  enabled: true
  go_package_prefix:
    default: github.com/example/repository/gen/proto/go
plugins:
  - name: go
    out: gen/proto/go
    opt: paths=source_relative
  - name: validate-go
    out: gen/proto/go
    opt: paths=source_relative
```

```sh
$ buf generate --include-imports
```

The generated file will contain the following:
```go
package examplev1

import (
	errors "errors"
	fmt "fmt"
	v1alpha1 "github.com/example/repository/gen/proto/go/buf/alpha/validate/v1alpha1"
	cel "github.com/google/cel-go/cel"
	proto "google.golang.org/protobuf/proto"
)

// Validate validates this message according to the given CEL expression, where 'this' is this instance.
//
//  has(this.key) && has(this.value)
func (x *CreateObjectRequest) Validate() error {
	return validateCreateObjectRequest(x)
}

var validateCreateObjectRequest func(*CreateObjectRequest) error

func init() {
	defaultMessage := &CreateObjectRequest{}
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

	validateCreateObjectRequest = func(x *CreateObjectRequest) error {
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
			return errors.New(stringVal)
		}
		return errors.New("CreateObjectRequest is invalid; see the message definition for details")
	}
}
```

There's several moving parts here, but the implementation can be described simply by the following:
  * The `init` hook is used to compile the CEL expression into an AST, and initialize the underlying implementation
    of the `bufvalidate.Validator` interface.
  * At runtime, evaluation is cheap - the function pointer is prepared upfront and any call to `.Validate()` simply
    runs the `program.Eval` for the CEL expression defined on the given `message`.

## Custom error messages

If you adapt your CEL expression to return a `string`, it will be used as the error message
returned from the `bufvalidate.Validator` implementation. For example,

```protobuf
syntax = "proto3";

package example.v1;

import "buf/alpha/validate/v1alpha1/validate.proto";

message GetObjectRequest {
  option (buf.alpha.validate.v1alpha1.expr) = {
    expression: "has(this.id) ? 'true' : 'GetObjectRequest must have a non-empty id'"
  };
  string id = 1;
}
```

Note that the `'true'` and `'false'` string values are equivalent to their `bool` equivalent
so that the ternary operator (`?`) is compatible with the CEL type system.

## Future work

### Generated fuzz testing

The `protoc-gen-validate-go` plugin performs lightweight validation to ensure that the
CEL expression is valid with respect to the `message` that you're validating. However,
the best it can do is verify that the expression is valid CEL, and that the default value
of the message yields an expected value (i.e. a `string` or a `bool`).

Fortunately, with [Go Fuzzing](https://go.dev/doc/fuzz), it's easy for the `protoc-gen-validate-go`
plugin to _also_ generate a valid fuzz test that exercises a larger input corpus for the `bufvalidate.Validator`
implementation.

For example, the following `_test.go` file could be generated alongside the `.validate.pb.go` file:
```go
package examplev1_test

func FuzzCreateObjectRequestValidate(f *testing.F) {
  f.Fuzz(func(t *testing.T, ...) {
    ...
  }
}
```

With this, you can treat the generated `bufvalidate.Validator` implementation just like your other
hand-written business logic - it's continuously tested alongside everything else with `go test`.

### Other language support

The approach taken here depends on a CEL interpreter. Today, there is a functional interpreter for
[Go](https://github.com/google/cel-go) and [C++](https://github.com/google/cel-cpp), but other languages
are not yet supported (nor is it clear if they ever will be).

Fortunately, C++ is a good choice for compatibility with WebAssembly ([WASM](https://webassembly.org)).
This means that the CEL interpreter evaluation could be implemented in C++ and compiled into a `.wasm`
module. Any language that operates well with WASM (JavaScript, TypeScript, Rust, etc) would be able to
leverage this solution.
