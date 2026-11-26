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
	"slices"

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

var celVariableNames = []string{"this", "rule", "rules"}

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

func (e celExpressionInfo) celDeclares(name string) bool {
	switch name {
	case "rule", "rules":
		return e.predefined
	}
	return true
}

func (e celExpressionInfo) celVariableIRTypes() map[string]ir.Type {
	vars := map[string]ir.Type{}
	for _, name := range celVariableNames {
		if irType, _ := e.celVariable(name); !irType.IsZero() {
			vars[name] = irType
		}
	}
	return vars
}

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
		fileProto.Extension = celDeleteUntypedFields(fileProto.Extension)
		for _, messageProto := range fileProto.MessageType {
			celDeleteUntypedMessageFields(messageProto)
		}
		fileDescriptor, err := options.New(fileProto, resolver)
		if err != nil {
			continue
		}
		if err := files.RegisterFile(fileDescriptor); err != nil {
			continue
		}
	}
	return files, nil
}

// celDeleteUntypedMessageFields deletes the fields and extensions of
// messageProto and its nested messages that have no type, which fdp writes for
// a field whose type does not resolve and protodesc rejects.
func celDeleteUntypedMessageFields(messageProto *descriptorpb.DescriptorProto) {
	messageProto.Field = celDeleteUntypedFields(messageProto.Field)
	messageProto.Extension = celDeleteUntypedFields(messageProto.Extension)
	for _, nestedProto := range messageProto.NestedType {
		celDeleteUntypedMessageFields(nestedProto)
	}
}

func celDeleteUntypedFields(fieldProtos []*descriptorpb.FieldDescriptorProto) []*descriptorpb.FieldDescriptorProto {
	return slices.DeleteFunc(fieldProtos, func(fieldProto *descriptorpb.FieldDescriptorProto) bool {
		return fieldProto.Type == nil && fieldProto.TypeName == nil
	})
}

type celWKTExcluder struct{}

func (celWKTExcluder) Exclude(file *ir.File) bool {
	return datawkt.Exists(file.Path())
}

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
