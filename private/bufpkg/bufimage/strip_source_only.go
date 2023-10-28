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

package bufimage

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func stripSourceOnlyOptions[M proto.Message](options M) (M, error) {
	optionsRef := options.ProtoReflect()
	// See if there are any options to strip.
	var found bool
	var err error
	optionsRef.Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		fieldOpts, ok := field.Options().(*descriptorpb.FieldOptions)
		if !ok {
			err = fmt.Errorf("field options is unexpected type: got %T, want %T", field.Options(), fieldOpts)
			return false
		}
		if fieldOpts.GetRetention() == descriptorpb.FieldOptions_RETENTION_SOURCE {
			found = true
			return false
		}
		return true
	})
	var zero M
	if err != nil {
		return zero, err
	}
	if !found {
		return options, nil
	}

	// There is at least one. So we need to make a copy that does not have those options.
	newOptions := optionsRef.New()
	ret, ok := newOptions.Interface().(M)
	if !ok {
		return zero, fmt.Errorf("creating new message of same type resulted in unexpected type; got %T, want %T", newOptions.Interface(), zero)
	}
	optionsRef.Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		fieldOpts, ok := field.Options().(*descriptorpb.FieldOptions)
		if !ok {
			err = fmt.Errorf("field options is unexpected type: got %T, want %T", field.Options(), fieldOpts)
			return false
		}
		if fieldOpts.GetRetention() != descriptorpb.FieldOptions_RETENTION_SOURCE {
			newOptions.Set(field, val)
		}
		return true
	})
	if err != nil {
		return zero, err
	}
	return ret, nil
}

func stripSourceOnlyOptionsFromFile(file *descriptorpb.FileDescriptorProto) (*descriptorpb.FileDescriptorProto, error) {
	var dirty bool
	newOpts, err := stripSourceOnlyOptions(file.Options)
	if err != nil {
		return nil, err
	}
	if newOpts != file.Options {
		dirty = true
	}
	newMsgs, changed, err := updateAll(file.MessageType, stripSourceOnlyOptionsFromMessage)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newEnums, changed, err := updateAll(file.EnumType, stripSourceOnlyOptionsFromEnum)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newExts, changed, err := updateAll(file.Extension, stripSourceOnlyOptionsFromField)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newSvcs, changed, err := updateAll(file.Service, stripSourceOnlyOptionsFromService)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}

	if !dirty {
		return file, nil
	}

	newFile, err := shallowCopy(file)
	if err != nil {
		return nil, err
	}
	newFile.Options = newOpts
	newFile.MessageType = newMsgs
	newFile.EnumType = newEnums
	newFile.Extension = newExts
	newFile.Service = newSvcs
	return newFile, nil
}

func stripSourceOnlyOptionsFromMessage(msg *descriptorpb.DescriptorProto) (*descriptorpb.DescriptorProto, error) {
	var dirty bool
	newOpts, err := stripSourceOnlyOptions(msg.Options)
	if err != nil {
		return nil, err
	}
	if newOpts != msg.Options {
		dirty = true
	}
	newFields, changed, err := updateAll(msg.Field, stripSourceOnlyOptionsFromField)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newOneofs, changed, err := updateAll(msg.OneofDecl, stripSourceOnlyOptionsFromOneof)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newExtRanges, changed, err := updateAll(msg.ExtensionRange, stripSourceOnlyOptionsFromExtensionRange)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newMsgs, changed, err := updateAll(msg.NestedType, stripSourceOnlyOptionsFromMessage)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newEnums, changed, err := updateAll(msg.EnumType, stripSourceOnlyOptionsFromEnum)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newExts, changed, err := updateAll(msg.Extension, stripSourceOnlyOptionsFromField)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}

	if !dirty {
		return msg, nil
	}

	newMsg, err := shallowCopy(msg)
	if err != nil {
		return nil, err
	}
	newMsg.Options = newOpts
	newMsg.Field = newFields
	newMsg.OneofDecl = newOneofs
	newMsg.ExtensionRange = newExtRanges
	newMsg.NestedType = newMsgs
	newMsg.EnumType = newEnums
	newMsg.Extension = newExts
	return newMsg, nil
}

func stripSourceOnlyOptionsFromField(field *descriptorpb.FieldDescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	newOpts, err := stripSourceOnlyOptions(field.Options)
	if err != nil {
		return nil, err
	}
	if newOpts == field.Options {
		return field, nil
	}
	newField, err := shallowCopy(field)
	if err != nil {
		return nil, err
	}
	newField.Options = newOpts
	return newField, nil
}

func stripSourceOnlyOptionsFromOneof(oneof *descriptorpb.OneofDescriptorProto) (*descriptorpb.OneofDescriptorProto, error) {
	newOpts, err := stripSourceOnlyOptions(oneof.Options)
	if err != nil {
		return nil, err
	}
	if newOpts == oneof.Options {
		return oneof, nil
	}
	newOneof, err := shallowCopy(oneof)
	if err != nil {
		return nil, err
	}
	newOneof.Options = newOpts
	return newOneof, nil
}

func stripSourceOnlyOptionsFromExtensionRange(extRange *descriptorpb.DescriptorProto_ExtensionRange) (*descriptorpb.DescriptorProto_ExtensionRange, error) {
	newOpts, err := stripSourceOnlyOptions(extRange.Options)
	if err != nil {
		return nil, err
	}
	if newOpts == extRange.Options {
		return extRange, nil
	}
	newExtRange, err := shallowCopy(extRange)
	if err != nil {
		return nil, err
	}
	newExtRange.Options = newOpts
	return newExtRange, nil
}

func stripSourceOnlyOptionsFromEnum(enum *descriptorpb.EnumDescriptorProto) (*descriptorpb.EnumDescriptorProto, error) {
	var dirty bool
	newOpts, err := stripSourceOnlyOptions(enum.Options)
	if err != nil {
		return nil, err
	}
	if newOpts != enum.Options {
		dirty = true
	}
	newVals, changed, err := updateAll(enum.Value, stripSourceOnlyOptionsFromEnumValue)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}

	if !dirty {
		return enum, nil
	}

	newEnum, err := shallowCopy(enum)
	if err != nil {
		return nil, err
	}
	newEnum.Options = newOpts
	newEnum.Value = newVals
	return newEnum, nil
}

func stripSourceOnlyOptionsFromEnumValue(enumVal *descriptorpb.EnumValueDescriptorProto) (*descriptorpb.EnumValueDescriptorProto, error) {
	newOpts, err := stripSourceOnlyOptions(enumVal.Options)
	if err != nil {
		return nil, err
	}
	if newOpts == enumVal.Options {
		return enumVal, nil
	}
	newEnumVal, err := shallowCopy(enumVal)
	if err != nil {
		return nil, err
	}
	newEnumVal.Options = newOpts
	return newEnumVal, nil
}

func stripSourceOnlyOptionsFromService(svc *descriptorpb.ServiceDescriptorProto) (*descriptorpb.ServiceDescriptorProto, error) {
	var dirty bool
	newOpts, err := stripSourceOnlyOptions(svc.Options)
	if err != nil {
		return nil, err
	}
	if newOpts != svc.Options {
		dirty = true
	}
	newMethods, changed, err := updateAll(svc.Method, stripSourceOnlyOptionsFromMethod)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}

	if !dirty {
		return svc, nil
	}

	newSvc, err := shallowCopy(svc)
	if err != nil {
		return nil, err
	}
	newSvc.Options = newOpts
	newSvc.Method = newMethods
	return newSvc, nil
}

func stripSourceOnlyOptionsFromMethod(method *descriptorpb.MethodDescriptorProto) (*descriptorpb.MethodDescriptorProto, error) {
	newOpts, err := stripSourceOnlyOptions(method.Options)
	if err != nil {
		return nil, err
	}
	if newOpts == method.Options {
		return method, nil
	}
	newMethod, err := shallowCopy(method)
	if err != nil {
		return nil, err
	}
	newMethod.Options = newOpts
	return newMethod, nil
}

func shallowCopy[M proto.Message](msg M) (M, error) {
	msgRef := msg.ProtoReflect()
	other := msgRef.New()
	ret, ok := other.Interface().(M)
	if !ok {
		return ret, fmt.Errorf("creating new message of same type resulted in unexpected type; got %T, want %T", other.Interface(), ret)
	}
	msgRef.Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		other.Set(field, val)
		return true
	})
	return ret, nil
}

// updateAll applies the given function to each element in the given slice. It
// returns the new slice and a bool indicating whether anything was actually
// changed. If the second value is false, then the returned slice is the same
// slice as the input slice. Usually, T is a pointer type, in which case the
// given updateFunc should NOT mutate the input value. Instead, it should return
// the input value if only if there is no update needed. If a mutation is needed,
// it should return a new value.
func updateAll[T comparable](slice []T, updateFunc func(T) (T, error)) ([]T, bool, error) {
	var updated []T // initialized lazily, only when/if a copy is needed
	for i, item := range slice {
		newItem, err := updateFunc(item)
		if err != nil {
			return nil, false, err
		}
		if updated != nil {
			updated[i] = newItem
		} else if newItem != item {
			updated = make([]T, len(slice))
			copy(updated[:i], slice)
			updated[i] = newItem
		}
	}
	if updated != nil {
		return updated, true, nil
	}
	return slice, false, nil
}
