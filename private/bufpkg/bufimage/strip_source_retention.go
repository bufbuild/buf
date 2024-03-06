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

func stripSourceRetentionOptionsFromFile(file *descriptorpb.FileDescriptorProto) (*descriptorpb.FileDescriptorProto, error) {
	var dirty bool
	newOpts, err := stripSourceRetentionOptions(file.Options)
	if err != nil {
		return nil, err
	}
	if newOpts != file.Options {
		dirty = true
	}
	newMsgs, changed, err := updateAll(file.MessageType, stripSourceRetentionOptionsFromMessage)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newEnums, changed, err := updateAll(file.EnumType, stripSourceRetentionOptionsFromEnum)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newExts, changed, err := updateAll(file.Extension, stripSourceRetentionOptionsFromField)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newSvcs, changed, err := updateAll(file.Service, stripSourceRetentionOptionsFromService)
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

func stripSourceRetentionOptions[M proto.Message](options M) (M, error) {
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

func stripSourceRetentionOptionsFromMessage(msg *descriptorpb.DescriptorProto) (*descriptorpb.DescriptorProto, error) {
	var dirty bool
	newOpts, err := stripSourceRetentionOptions(msg.Options)
	if err != nil {
		return nil, err
	}
	if newOpts != msg.Options {
		dirty = true
	}
	newFields, changed, err := updateAll(msg.Field, stripSourceRetentionOptionsFromField)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newOneofs, changed, err := updateAll(msg.OneofDecl, stripSourceRetentionOptionsFromOneof)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newExtRanges, changed, err := updateAll(msg.ExtensionRange, stripSourceRetentionOptionsFromExtensionRange)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newMsgs, changed, err := updateAll(msg.NestedType, stripSourceRetentionOptionsFromMessage)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newEnums, changed, err := updateAll(msg.EnumType, stripSourceRetentionOptionsFromEnum)
	if err != nil {
		return nil, err
	}
	if changed {
		dirty = true
	}
	newExts, changed, err := updateAll(msg.Extension, stripSourceRetentionOptionsFromField)
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

func stripSourceRetentionOptionsFromField(field *descriptorpb.FieldDescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	newOpts, err := stripSourceRetentionOptions(field.Options)
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

func stripSourceRetentionOptionsFromOneof(oneof *descriptorpb.OneofDescriptorProto) (*descriptorpb.OneofDescriptorProto, error) {
	newOpts, err := stripSourceRetentionOptions(oneof.Options)
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

func stripSourceRetentionOptionsFromExtensionRange(extRange *descriptorpb.DescriptorProto_ExtensionRange) (*descriptorpb.DescriptorProto_ExtensionRange, error) {
	newOpts, err := stripSourceRetentionOptions(extRange.Options)
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

func stripSourceRetentionOptionsFromEnum(enum *descriptorpb.EnumDescriptorProto) (*descriptorpb.EnumDescriptorProto, error) {
	var dirty bool
	newOpts, err := stripSourceRetentionOptions(enum.Options)
	if err != nil {
		return nil, err
	}
	if newOpts != enum.Options {
		dirty = true
	}
	newVals, changed, err := updateAll(enum.Value, stripSourceRetentionOptionsFromEnumValue)
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

func stripSourceRetentionOptionsFromEnumValue(enumVal *descriptorpb.EnumValueDescriptorProto) (*descriptorpb.EnumValueDescriptorProto, error) {
	newOpts, err := stripSourceRetentionOptions(enumVal.Options)
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

func stripSourceRetentionOptionsFromService(svc *descriptorpb.ServiceDescriptorProto) (*descriptorpb.ServiceDescriptorProto, error) {
	var dirty bool
	newOpts, err := stripSourceRetentionOptions(svc.Options)
	if err != nil {
		return nil, err
	}
	if newOpts != svc.Options {
		dirty = true
	}
	newMethods, changed, err := updateAll(svc.Method, stripSourceRetentionOptionsFromMethod)
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

func stripSourceRetentionOptionsFromMethod(method *descriptorpb.MethodDescriptorProto) (*descriptorpb.MethodDescriptorProto, error) {
	newOpts, err := stripSourceRetentionOptions(method.Options)
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
