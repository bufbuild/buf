// Copyright 2020-2026 Buf Technologies, Inc.
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

package buflsp

import (
	"log/slog"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/standard/xlog/xslog"
	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/protocompile/experimental/fdp"
	"github.com/bufbuild/protocompile/experimental/ir"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// celVariableNames are the protovalidate variables bound per expression. The
// global "now" is declared by the base environment.
//
// https://protovalidate.com/reference/cel_extensions/#variables
var celVariableNames = []string{"this", "rule", "rules"}

// celVariable returns the IR element type of the protovalidate variable name
// in the expression, and the member whose cardinality applies to it. Either
// may be zero: message-level `this` has no member, and the `this` of a
// predefined rule is not a proto type (see [celExpressionInfo.celEnv]).
func (e celExpressionInfo) celVariable(name string) (ir.Type, ir.Member) {
	if e.predefined {
		switch name {
		case "rule":
			return e.irMember.Element(), e.irMember
		case "rules":
			return e.irMember.Container(), ir.Member{}
		}
		return ir.Type{}, ir.Member{}
	}
	if name == "this" {
		return e.thisIRType, e.thisMember
	}
	return ir.Type{}, ir.Member{}
}

// celVariableIRTypes returns the IR element types of the protovalidate
// variables in the expression that have one, keyed by name.
func (e celExpressionInfo) celVariableIRTypes() map[string]ir.Type {
	vars := map[string]ir.Type{}
	for _, name := range celVariableNames {
		if irType, _ := e.celVariable(name); !irType.IsZero() {
			vars[name] = irType
		}
	}
	return vars
}

// celEnv returns base extended with the types of file (see [file.celEnv]) and
// the protovalidate variables in the expression. Variables that cannot be
// declared are left out.
func (e celExpressionInfo) celEnv(file *file, base *cel.Env) *cel.Env {
	base = file.celEnv(base)
	var options []cel.EnvOption
	if e.predefined {
		if thisType := celStandardRulesTypes[e.irMember.Container().FullName()]; thisType != nil {
			options = append(options, cel.Variable("this", thisType))
		}
	}
	for _, name := range celVariableNames {
		if celType := celMemberType(e.celVariable(name)); celType != nil {
			options = append(options, cel.Variable(name, celType))
		}
	}
	if len(options) == 0 {
		return base
	}
	env, err := base.Extend(options...)
	if err != nil {
		return base
	}
	return env
}

// celMemberType returns the CEL type of a value with element type irType and
// the cardinality of member, or nil if it has no CEL type.
func celMemberType(irType ir.Type, member ir.Member) *types.Type {
	switch {
	case member.IsMap():
		return celMapEntryToCELType(irType)
	case member.IsRepeated():
		if elemType := celProtoTypeToExprType(irType); elemType != nil {
			return types.NewListType(elemType)
		}
		return nil
	default:
		return celProtoTypeToExprType(irType)
	}
}

// celStandardRulesTypes are the CEL types of `this` in a predefined rule,
// keyed by the buf.validate.*Rules message the rule extends.
var celStandardRulesTypes = map[ir.FullName]*types.Type{
	celMessageName(&validate.AnyRules{}):       types.AnyType,
	celMessageName(&validate.BoolRules{}):      types.BoolType,
	celMessageName(&validate.BytesRules{}):     types.BytesType,
	celMessageName(&validate.StringRules{}):    types.StringType,
	celMessageName(&validate.Int32Rules{}):     types.IntType,
	celMessageName(&validate.Int64Rules{}):     types.IntType,
	celMessageName(&validate.SInt32Rules{}):    types.IntType,
	celMessageName(&validate.SInt64Rules{}):    types.IntType,
	celMessageName(&validate.SFixed32Rules{}):  types.IntType,
	celMessageName(&validate.SFixed64Rules{}):  types.IntType,
	celMessageName(&validate.EnumRules{}):      types.IntType,
	celMessageName(&validate.UInt32Rules{}):    types.UintType,
	celMessageName(&validate.UInt64Rules{}):    types.UintType,
	celMessageName(&validate.Fixed32Rules{}):   types.UintType,
	celMessageName(&validate.Fixed64Rules{}):   types.UintType,
	celMessageName(&validate.FloatRules{}):     types.DoubleType,
	celMessageName(&validate.DoubleRules{}):    types.DoubleType,
	celMessageName(&validate.MapRules{}):       types.NewMapType(types.DynType, types.DynType),
	celMessageName(&validate.RepeatedRules{}):  types.NewListType(types.DynType),
	celMessageName(&validate.DurationRules{}):  types.DurationType,
	celMessageName(&validate.TimestampRules{}): types.TimestampType,
	celMessageName(&validate.FieldMaskRules{}): types.NewObjectType(string(celMessageName(&fieldmaskpb.FieldMask{}))),
}

// celEnv returns base extended with the message and enum types of the file
// and its imports, so that expressions selecting their fields type-check.
// It returns base if the file has no IR or its descriptors cannot be built.
//
// The environment is cached until the file's IR changes.
func (f *file) celEnv(base *cel.Env) *cel.Env {
	if f.celTypesEnv != nil {
		return f.celTypesEnv
	}
	f.celTypesEnv = base
	if f.ir == nil {
		return base
	}
	files, err := celFileTypes(f.ir)
	if err != nil {
		f.lsp.logger.Debug("failed to build CEL types for file", slog.String("uri", string(f.uri)), xslog.ErrorAttr(err))
		return base
	}
	env, err := base.Extend(cel.TypeDescs(files))
	if err != nil {
		f.lsp.logger.Debug("failed to register CEL types for file", slog.String("uri", string(f.uri)), xslog.ErrorAttr(err))
		return base
	}
	f.celTypesEnv = env
	return env
}

// celFileTypes returns the descriptors of irFile and its imports.
//
// References that do not resolve, as in a file with errors, become
// placeholders rather than failing the file. The Well-Known Types are left
// out of the descriptor set and resolve from [protoregistry.GlobalFiles].
func celFileTypes(irFile *ir.File) (*protoregistry.Files, error) {
	data, err := fdp.DescriptorSetBytes([]*ir.File{irFile}, fdp.ExcludeFiles(celWKTExcluder{}))
	if err != nil {
		return nil, err
	}
	// Unmarshaling parses the option values that fdp writes as unknown fields,
	// such as EnumOptions.allow_alias, which protodesc validates against.
	fds := new(descriptorpb.FileDescriptorSet)
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, fds); err != nil {
		return nil, err
	}
	files := new(protoregistry.Files)
	resolver := celFileResolver{files}
	options := protodesc.FileOptions{AllowUnresolvable: true}
	for _, fileProto := range fds.GetFile() {
		fileDescriptor, err := options.New(fileProto, resolver)
		if err != nil {
			// The types of a file that fails to build are left out.
			continue
		}
		if err := files.RegisterFile(fileDescriptor); err != nil {
			continue
		}
	}
	return files, nil
}

// celWKTExcluder excludes the Well-Known Types from descriptor sets.
type celWKTExcluder struct{}

func (celWKTExcluder) Exclude(file *ir.File) bool {
	return datawkt.Exists(file.Path())
}

// celFileResolver resolves descriptors from files, then from
// [protoregistry.GlobalFiles].
type celFileResolver struct {
	files *protoregistry.Files
}

func (r celFileResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	if fileDescriptor, err := r.files.FindFileByPath(path); err == nil {
		return fileDescriptor, nil
	}
	return protoregistry.GlobalFiles.FindFileByPath(path)
}

func (r celFileResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	if descriptor, err := r.files.FindDescriptorByName(name); err == nil {
		return descriptor, nil
	}
	return protoregistry.GlobalFiles.FindDescriptorByName(name)
}
