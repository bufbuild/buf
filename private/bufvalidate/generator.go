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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	validatev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/validate/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	celPackage      = protogen.GoImportPath("github.com/google/cel-go/cel")
	errorsPackage   = protogen.GoImportPath("errors")
	fmtPackage      = protogen.GoImportPath("fmt")
	protoPackage    = protogen.GoImportPath("google.golang.org/protobuf/proto")
	strconvPackage  = protogen.GoImportPath("strconv")
	validatePackage = protogen.GoImportPath("github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/validate/v1alpha1")
)

type generator struct {
	plugin *protogen.Plugin
}

func newGenerator(
	plugin *protogen.Plugin,
) *generator {
	return &generator{
		plugin: plugin,
	}
}

func (g *generator) Generate(ctx context.Context) error {
	image, err := bufimage.NewImageForCodeGeneratorRequest(g.plugin.Request)
	if err != nil {
		return err
	}
	for _, file := range g.plugin.Files {
		if !file.Generate {
			continue
		}
		for _, message := range file.Messages {
			options, ok := message.Desc.Options().(*descriptorpb.MessageOptions)
			if !ok {
				return fmt.Errorf("expected *descriptorpb.MessageOptions, but got %T", message.Desc.Options())
			}
			if !proto.HasExtension(options, validatev1alpha1.E_Expr) {
				continue
			}
			expr, ok := proto.GetExtension(options, validatev1alpha1.E_Expr).(*validatev1alpha1.Expr)
			if !ok {
				return fmt.Errorf("expected string, but got %T", expr)
			}
			if err := validateExpr(ctx, image, message, expr); err != nil {
				return fmt.Errorf("invalid validation option: %v", err)
			}
			if err := g.generateFile(image, message, file, expr); err != nil {
				return fmt.Errorf("failed to generate file: %v", err)
			}
		}
	}
	return nil
}

// generateFile generates the code required to validate the given message.
// Every message gets its own .pb.validate.go file with its own init() function
// hook used to parse the CEL AST at build time (rather than on the hot path).
func (g *generator) generateFile(
	image bufimage.Image,
	message *protogen.Message,
	file *protogen.File,
	expr *validatev1alpha1.Expr,
) error {
	f := g.plugin.NewGeneratedFile(
		file.GeneratedFilenamePrefix+"."+stringutil.ToLowerSnakeCase(string(message.Desc.Name()))+".validate.pb.go",
		file.GoImportPath,
	)
	f.P("package ", file.GoPackageName)
	f.P()
	if err := generateValidateMethod(f, image, message, expr); err != nil {
		return err
	}
	f.P()
	if err := generateInit(f, image, message, file); err != nil {
		return err
	}
	return nil
}

// generateInit generates the code required to setup the CEL AST in an init() function
// hook. The init() hook is used so that the AST is parsed at build time rather than on
// the hot path.
//
// For example,
//
//  var validateFoo = func(*Foo) error
//
//  func init() {
//    options := Foo{}.ProtoReflect().Descriptor().Options()
//    expr, ok := proto.GetExtension(options, validatev1alpha1.E_Expr).(*validatev1alpha1.Expr)
//    if !ok {
//      panic(fmt.Errorf("expected string, but got %T", expr))
//    }
//    env, err := cel.NewEnv(cel.TypeDescs(File_example_v1_example_proto))
//    if err != nil {
//      panic(err)
//    }
//    ast, issues := env.Compile(expr.GetExpression())
//    if issues.Err() != nil {
//      panic(issues.Err())
//    }
//    program, err := env.Program(ast)
//    if err != nil {
//      panic(err)
//    }
//
//    validateFoo = func(x *Foo) error {
//      val, _, err := program.Eval(
//        map[string]interface{}{
//          "this": x,
//        },
//      )
//      if val == nil {
//        return err
//      }
//      value := val.Value()
//      if boolVal, ok := value.(bool); ok && boolVal {
//        return nil
//      }
//      if stringVal, ok := value.(string); ok {
//        boolVal, ok := strconv.ParseBool(stringVal)
//        if !ok {
//          return errors.New(stringVal)
//        }
//        if boolVal {
//          return nil
//        }
//      }
//      return errors.New("Foo is invalid; see the message definition for details")
//    }
//  }
//
func generateInit(
	f *protogen.GeneratedFile,
	image bufimage.Image,
	message *protogen.Message,
	file *protogen.File,
) error {
	f.P("var validate" + f.QualifiedGoIdent(message.GoIdent) + " func(*" + f.QualifiedGoIdent(message.GoIdent) + ") error")
	f.P()
	f.P("func init() {")
	f.P("defaultMessage := &", f.QualifiedGoIdent(message.GoIdent)+"{}")
	f.P("options := defaultMessage.ProtoReflect().Descriptor().Options()")
	f.P("expr, ok := " + f.QualifiedGoIdent(protoPackage.Ident("GetExtension")) + "(options, " + f.QualifiedGoIdent(validatePackage.Ident("E_Expr")) + ").(*" + f.QualifiedGoIdent(validatePackage.Ident("Expr")) + ")")
	f.P("if !ok {")
	f.P(`panic(` + f.QualifiedGoIdent(fmtPackage.Ident("Errorf")) + `("expected CEL expression, but got %T", expr))`)
	f.P("}")
	f.P("env, err := ", f.QualifiedGoIdent(celPackage.Ident("NewEnv"))+"(")
	f.P(f.QualifiedGoIdent(celPackage.Ident("Variable")) + "(")
	f.P(`"this",`)
	f.P(f.QualifiedGoIdent(celPackage.Ident("ObjectType")) + "(string(defaultMessage.ProtoReflect().Descriptor().FullName())),")
	f.P("),")
	f.P(f.QualifiedGoIdent(celPackage.Ident("TypeDescs")) + "(" + f.QualifiedGoIdent(file.GoDescriptorIdent) + "),")
	f.P(")")
	f.P("if err != nil {")
	f.P("panic(err)")
	f.P("}")
	f.P("ast, issues := env.Compile(expr.GetExpression())")
	f.P("if err := issues.Err(); err != nil {")
	f.P("panic(err)")
	f.P("}")
	f.P("program, err := env.Program(ast)")
	f.P("if err != nil {")
	f.P("panic(err)")
	f.P("}")
	f.P()
	f.P("validate" + f.QualifiedGoIdent(message.GoIdent) + " = func(x *" + f.QualifiedGoIdent(message.GoIdent) + ") error {")
	f.P("val, _, err := program.Eval(")
	f.P("map[string]interface{}{")
	f.P(`"this": x,`)
	f.P("},")
	f.P(")")
	f.P("if val == nil {")
	f.P("return err")
	f.P("}")
	f.P("value := val.Value()")
	f.P("if boolVal, ok := value.(bool); ok && boolVal {")
	f.P("return nil")
	f.P("}")
	f.P("if stringVal, ok := value.(string); ok {")
	f.P("boolVal, err := " + f.QualifiedGoIdent(strconvPackage.Ident("ParseBool")) + "(stringVal)")
	f.P("if err != nil {")
	f.P("return " + f.QualifiedGoIdent(errorsPackage.Ident("New")) + "(stringVal)")
	f.P("}")
	f.P("if boolVal {")
	f.P("return nil")
	f.P("}")
	f.P("}")
	f.P(`return ` + f.QualifiedGoIdent(errorsPackage.Ident("New")) + `("` + f.QualifiedGoIdent(message.GoIdent) + ` is invalid; see the message definition for details")`)
	f.P("}")
	f.P("}")
	return nil
}

// generateValidateMethod generates the .Validate() method on the Go type for the given message
// so that it implements the Validator interface.
//
// The method uses the CEL interpreter initialized in the init() hook to determine whether or not
// the type's values represent a valid structure.
//
// For example,
//
//  func (f *Foo) Validate() error {
//    return validateFoo(f)
//  }
//
func generateValidateMethod(
	f *protogen.GeneratedFile,
	image bufimage.Image,
	message *protogen.Message,
	expr *validatev1alpha1.Expr,
) error {
	f.P("// Validate validates this message according to the given CEL expression, where 'this' is this instance.")
	f.P("//")
	f.P("//  " + expr.GetExpression())
	f.P("func (x *", f.QualifiedGoIdent(message.GoIdent), ") Validate() error {")
	f.P("return validate" + f.QualifiedGoIdent(message.GoIdent) + "(x)")
	f.P("}")
	return nil
}

// validateExpr validates that the given *validatev1alpha1.Expr is well-formed.
//
// Note that this validation is a best-effort; we can only validate that the
// default value of the message produces a valid result. The user will need
// to incorporate fuzz testing for their validation code to confirm that it's
// working as expected against a larger corpus of input values.
func validateExpr(
	ctx context.Context,
	image bufimage.Image,
	message *protogen.Message,
	expr *validatev1alpha1.Expr,
) error {
	if len(expr.GetExpression()) == 0 {
		return errors.New("CEL expression must be non-empty")
	}
	dynamicMessage, err := bufreflect.NewMessage(ctx, image, string(message.Desc.FullName()))
	if err != nil {
		return err
	}
	env, err := cel.NewEnv(
		cel.Types(dynamicMessage),
		cel.Variable(
			"this",
			cel.ObjectType(string(message.Desc.FullName())),
		),
	)
	if err != nil {
		return err
	}
	ast, issues := env.Compile(expr.GetExpression())
	if issues.Err() != nil {
		return issues.Err()
	}
	program, err := env.Program(ast)
	if err != nil {
		return err
	}
	// The return values here are a little different than what we're
	// used to with idiomatic Go - a non-nil error value can actually
	// represent a valid evaluation (one that returns an CEL error
	// value).
	//
	// In this case, the value of ref.Val actually tells us
	// if the operation succeeded.
	val, _, err := program.Eval(
		map[string]interface{}{
			"this": dynamicMessage,
		},
	)
	if val == nil {
		return err
	}
	typeName := val.Type().TypeName()
	if typeName == "bool" || typeName == "string" {
		return nil
	}
	return fmt.Errorf("CEL expression must return a bool or string value, but got %s", typeName)
}
