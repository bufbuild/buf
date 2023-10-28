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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestStripSourceOnlyOptions(t *testing.T) {
	t.Parallel()
	optsFileProto := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("opts.proto"),
		Package:    proto.String("foo.bar"),
		Dependency: []string{"google/protobuf/descriptor.proto"},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: proto.String(".google.protobuf.FileOptions"),
				Name:     proto.String("no_retention"),
				Number:   proto.Int32(10000),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				// No option
			},
			{
				Extendee: proto.String(".google.protobuf.FileOptions"),
				Name:     proto.String("unknown_retention"),
				Number:   proto.Int32(10001),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Options: &descriptorpb.FieldOptions{
					Retention: descriptorpb.FieldOptions_RETENTION_UNKNOWN.Enum(),
				},
			},
			{
				Extendee: proto.String(".google.protobuf.FileOptions"),
				Name:     proto.String("runtime_retention"),
				Number:   proto.Int32(10002),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
				Options: &descriptorpb.FieldOptions{
					Retention: descriptorpb.FieldOptions_RETENTION_RUNTIME.Enum(),
				},
			},
			{
				Extendee: proto.String(".google.protobuf.FileOptions"),
				Name:     proto.String("source_retention"),
				Number:   proto.Int32(10003),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				Options: &descriptorpb.FieldOptions{
					Retention: descriptorpb.FieldOptions_RETENTION_SOURCE.Enum(),
				},
			},
		},
	}
	optsFile, err := protodesc.NewFile(optsFileProto, protoregistry.GlobalFiles)
	require.NoError(t, err)
	extNoRetention := dynamicpb.NewExtensionType(optsFile.Extensions().ByName("no_retention"))
	extUnknownRetention := dynamicpb.NewExtensionType(optsFile.Extensions().ByName("unknown_retention"))
	extRuntimeRetention := dynamicpb.NewExtensionType(optsFile.Extensions().ByName("runtime_retention"))
	extSourceRetention := dynamicpb.NewExtensionType(optsFile.Extensions().ByName("source_retention"))

	// Create a message with these options.
	options := (&descriptorpb.FileOptions{}).ProtoReflect()
	options.Set(extNoRetention.TypeDescriptor(), protoreflect.ValueOfString("abc"))
	listVal := extUnknownRetention.New().List()
	listVal.Append(protoreflect.ValueOfString("foo"))
	listVal.Append(protoreflect.ValueOfString("bar"))
	options.Set(extUnknownRetention.TypeDescriptor(), protoreflect.ValueOfList(listVal))
	options.Set(extRuntimeRetention.TypeDescriptor(), protoreflect.ValueOfBytes([]byte("xyz")))
	// The above will be retained, so create a copy now to serve as the expected result.
	optionsAfterStrip := proto.Clone(options.Interface())
	// The below option will get stripped because it's retention policy is source.
	listVal = extSourceRetention.New().List()
	listVal.Append(protoreflect.ValueOfInt32(123))
	listVal.Append(protoreflect.ValueOfInt32(-456))
	options.Set(extSourceRetention.TypeDescriptor(), protoreflect.ValueOfList(listVal))

	optionsMsg := options.Interface()
	actualOptionsAfterStrip, err := stripSourceOnlyOptions(optionsMsg)
	require.NoError(t, err)

	require.NotSame(t, actualOptionsAfterStrip, optionsMsg)
	require.Empty(t, cmp.Diff(optionsAfterStrip, actualOptionsAfterStrip, protocmp.Transform()))

	// If we do it again, there are no changes to made (since source-only options were
	// already stripped). So we should get back unmodified value.
	optionsMsg = actualOptionsAfterStrip
	actualOptionsAfterStrip, err = stripSourceOnlyOptions(optionsMsg)
	require.NoError(t, err)

	require.Same(t, actualOptionsAfterStrip, optionsMsg)
	require.Empty(t, cmp.Diff(optionsAfterStrip, actualOptionsAfterStrip, protocmp.Transform()))
}

func TestStripSourceOnlyOptionsFromFile(t *testing.T) {
	t.Parallel()
	makeCustomOptionSet := func(startTag int32, extendee string, prefix string, label descriptorpb.FieldDescriptorProto_Label) []*descriptorpb.FieldDescriptorProto {
		return []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: proto.String(extendee),
				Name:     proto.String(prefix + "no_retention"),
				Number:   proto.Int32(startTag),
				Label:    label.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				// No option
			},
			{
				Extendee: proto.String(extendee),
				Name:     proto.String(prefix + "unknown_retention"),
				Number:   proto.Int32(startTag + 1),
				Label:    label.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
				Options: &descriptorpb.FieldOptions{
					Retention: descriptorpb.FieldOptions_RETENTION_UNKNOWN.Enum(),
				},
			},
			{
				Extendee: proto.String(extendee),
				Name:     proto.String(prefix + "runtime_retention"),
				Number:   proto.Int32(startTag + 2),
				Label:    label.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
				Options: &descriptorpb.FieldOptions{
					Retention: descriptorpb.FieldOptions_RETENTION_RUNTIME.Enum(),
				},
			},
			{
				Extendee: proto.String(extendee),
				Name:     proto.String(prefix + "source_retention"),
				Number:   proto.Int32(startTag + 3),
				Label:    label.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				Options: &descriptorpb.FieldOptions{
					Retention: descriptorpb.FieldOptions_RETENTION_SOURCE.Enum(),
				},
			},
		}
	}
	makeCustomOptions := func(extendee string, prefix string) []*descriptorpb.FieldDescriptorProto {
		return append(
			makeCustomOptionSet(10000, extendee, prefix, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
			makeCustomOptionSet(20000, extendee, prefix+"rep_", descriptorpb.FieldDescriptorProto_LABEL_REPEATED)...,
		)
	}
	combineAll := func(exts ...[]*descriptorpb.FieldDescriptorProto) []*descriptorpb.FieldDescriptorProto {
		result := exts[0]
		for _, exts := range exts[1:] {
			result = append(result, exts...)
		}
		return result
	}
	optsFileProto := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("opts.proto"),
		Package:    proto.String("foo.bar"),
		Dependency: []string{"google/protobuf/descriptor.proto"},
		Extension: combineAll(
			makeCustomOptions(".google.protobuf.FileOptions", "file_"),
			makeCustomOptions(".google.protobuf.MessageOptions", "msg_"),
			makeCustomOptions(".google.protobuf.FieldOptions", "field_"),
			makeCustomOptions(".google.protobuf.OneofOptions", "oneof_"),
			makeCustomOptions(".google.protobuf.ExtensionRangeOptions", "extrange_"),
			makeCustomOptions(".google.protobuf.EnumOptions", "enum_"),
			makeCustomOptions(".google.protobuf.EnumValueOptions", "enumval_"),
			makeCustomOptions(".google.protobuf.ServiceOptions", "svc_"),
			makeCustomOptions(".google.protobuf.MethodOptions", "method_"),
		),
	}
	optsFile, err := protodesc.NewFile(optsFileProto, protoregistry.GlobalFiles)
	require.NoError(t, err)

	applyCustomOptionSet := func(all, retained protoreflect.Message, prefix protoreflect.Name, isList bool, file protoreflect.FileDescriptor) {
		extType := dynamicpb.NewExtensionType(file.Extensions().ByName(prefix + "no_retention"))
		var val protoreflect.Value
		if isList {
			listVal := extType.New().List()
			listVal.Append(protoreflect.ValueOfString("foo"))
			listVal.Append(protoreflect.ValueOfString("bar"))
			val = protoreflect.ValueOfList(listVal)
		} else {
			val = protoreflect.ValueOfString("abc")
		}
		all.Set(extType.TypeDescriptor(), val)
		retained.Set(extType.TypeDescriptor(), val)

		extType = dynamicpb.NewExtensionType(file.Extensions().ByName(prefix + "unknown_retention"))
		if isList {
			listVal := extType.New().List()
			listVal.Append(protoreflect.ValueOfBool(false))
			listVal.Append(protoreflect.ValueOfBool(true))
			val = protoreflect.ValueOfList(listVal)
		} else {
			val = protoreflect.ValueOfBool(true)
		}
		all.Set(extType.TypeDescriptor(), val)
		retained.Set(extType.TypeDescriptor(), val)

		extType = dynamicpb.NewExtensionType(file.Extensions().ByName(prefix + "runtime_retention"))
		if isList {
			listVal := extType.New().List()
			listVal.Append(protoreflect.ValueOfBytes([]byte{0, 1, 2, 3}))
			listVal.Append(protoreflect.ValueOfBytes([]byte{4, 5, 6, 7}))
			val = protoreflect.ValueOfList(listVal)
		} else {
			val = protoreflect.ValueOfBytes([]byte{0, 1, 2, 3})
		}
		all.Set(extType.TypeDescriptor(), val)
		retained.Set(extType.TypeDescriptor(), val)

		extType = dynamicpb.NewExtensionType(file.Extensions().ByName(prefix + "source_retention"))
		if isList {
			listVal := extType.New().List()
			listVal.Append(protoreflect.ValueOfInt32(123))
			listVal.Append(protoreflect.ValueOfInt32(-456))
			val = protoreflect.ValueOfList(listVal)
		} else {
			val = protoreflect.ValueOfInt32(123)
		}
		all.Set(extType.TypeDescriptor(), val)
		// don't set retained because this is a source-only option (won't be retained)
	}
	applyCustomOptions := func(message proto.Message, prefix protoreflect.Name, file protoreflect.FileDescriptor) (all, retained proto.Message) {
		allRef := message.ProtoReflect()
		strippedRef := proto.Clone(message).ProtoReflect()
		applyCustomOptionSet(allRef, strippedRef, prefix, false, file)
		applyCustomOptionSet(allRef, strippedRef, prefix+"rep_", true, file)
		return allRef.Interface(), strippedRef.Interface()
	}

	fileOpts, fileOptsStripped := applyCustomOptions(&descriptorpb.FileOptions{}, "file_", optsFile)
	msgOpts, msgOptsStripped := applyCustomOptions(&descriptorpb.MessageOptions{}, "msg_", optsFile)
	fieldOpts, fieldOptsStripped := applyCustomOptions(&descriptorpb.FieldOptions{}, "field_", optsFile)
	oneofOpts, oneofOptsStripped := applyCustomOptions(&descriptorpb.OneofOptions{}, "oneof_", optsFile)
	extRangeOpts, extRangeOptsStripped := applyCustomOptions(&descriptorpb.ExtensionRangeOptions{}, "extrange_", optsFile)
	enumOpts, enumOptsStripped := applyCustomOptions(&descriptorpb.EnumOptions{}, "enum_", optsFile)
	enumValOpts, enumValOptsStripped := applyCustomOptions(&descriptorpb.EnumValueOptions{}, "enumval_", optsFile)
	svcOpts, svcOptsStripped := applyCustomOptions(&descriptorpb.ServiceOptions{}, "svc_", optsFile)
	methodOpts, methodOptsStripped := applyCustomOptions(&descriptorpb.MethodOptions{}, "method_", optsFile)

	beforeFile := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("foo.proto"),
		Options: fileOpts.(*descriptorpb.FileOptions),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:    proto.String("Message"),
				Options: msgOpts.(*descriptorpb.MessageOptions),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:       proto.String("field"),
						Number:     proto.Int32(1),
						Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						JsonName:   proto.String("field"),
						Options:    fieldOpts.(*descriptorpb.FieldOptions),
						OneofIndex: proto.Int32(0),
					},
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{
						Name:    proto.String("oo"),
						Options: oneofOpts.(*descriptorpb.OneofOptions),
					},
				},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{
						Start:   proto.Int32(100),
						End:     proto.Int32(200),
						Options: extRangeOpts.(*descriptorpb.ExtensionRangeOptions),
					},
				},
			},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name:    proto.String("Enum"),
				Options: enumOpts.(*descriptorpb.EnumOptions),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{
						Name:    proto.String("ZERO"),
						Number:  proto.Int32(0),
						Options: enumValOpts.(*descriptorpb.EnumValueOptions),
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name:    proto.String("Service"),
				Options: svcOpts.(*descriptorpb.ServiceOptions),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("Do"),
						InputType:  proto.String(".Message"),
						OutputType: proto.String(".Message"),
						Options:    methodOpts.(*descriptorpb.MethodOptions),
					},
				},
			},
		},
	}

	// This one is the same as above, but uses the stripped option messages
	afterFile := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("foo.proto"),
		Options: fileOptsStripped.(*descriptorpb.FileOptions),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:    proto.String("Message"),
				Options: msgOptsStripped.(*descriptorpb.MessageOptions),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:       proto.String("field"),
						Number:     proto.Int32(1),
						Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						JsonName:   proto.String("field"),
						Options:    fieldOptsStripped.(*descriptorpb.FieldOptions),
						OneofIndex: proto.Int32(0),
					},
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{
						Name:    proto.String("oo"),
						Options: oneofOptsStripped.(*descriptorpb.OneofOptions),
					},
				},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{
						Start:   proto.Int32(100),
						End:     proto.Int32(200),
						Options: extRangeOptsStripped.(*descriptorpb.ExtensionRangeOptions),
					},
				},
			},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name:    proto.String("Enum"),
				Options: enumOptsStripped.(*descriptorpb.EnumOptions),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{
						Name:    proto.String("ZERO"),
						Number:  proto.Int32(0),
						Options: enumValOptsStripped.(*descriptorpb.EnumValueOptions),
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name:    proto.String("Service"),
				Options: svcOptsStripped.(*descriptorpb.ServiceOptions),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("Do"),
						InputType:  proto.String(".Message"),
						OutputType: proto.String(".Message"),
						Options:    methodOptsStripped.(*descriptorpb.MethodOptions),
					},
				},
			},
		},
	}

	actualStrippedFile, err := stripSourceOnlyOptionsFromFile(beforeFile)
	require.NoError(t, err)
	require.NotSame(t, actualStrippedFile, beforeFile)
	require.Empty(t, cmp.Diff(afterFile, actualStrippedFile, protocmp.Transform()))

	// If we repeat the operation, we get back the same descriptor unchanged because
	// it doesn't have any source-only options.
	doubleStrippedFile, err := stripSourceOnlyOptionsFromFile(actualStrippedFile)
	require.NoError(t, err)
	require.Same(t, doubleStrippedFile, actualStrippedFile)
	require.Empty(t, cmp.Diff(afterFile, doubleStrippedFile, protocmp.Transform()))
}

func TestUpdateAll(t *testing.T) {
	t.Parallel()

	errInvalid := errors.New("invalid value")
	updateFunc := func(i *int32) (*int32, error) {
		if i == nil {
			return proto.Int32(-1), nil
		}
		if *i <= -100 {
			return nil, errInvalid
		}
		if *i > 5 {
			return proto.Int32(*i * 2), nil
		}
		return i, nil
	}

	vals := []*int32{
		proto.Int32(0), proto.Int32(1), proto.Int32(2),
		proto.Int32(3), proto.Int32(4), proto.Int32(5),
		proto.Int32(6), proto.Int32(7), proto.Int32(8),
	}
	newVals, changed, err := updateAll(vals, updateFunc)
	require.NoError(t, err)
	require.True(t, changed)
	expected := []*int32{
		proto.Int32(0), proto.Int32(1), proto.Int32(2),
		proto.Int32(3), proto.Int32(4), proto.Int32(5),
		proto.Int32(12), proto.Int32(14), proto.Int32(16),
	}
	require.Equal(t, expected, newVals)

	vals = []*int32{
		nil, proto.Int32(1), proto.Int32(2),
		proto.Int32(3), proto.Int32(4), proto.Int32(5),
	}
	newVals, changed, err = updateAll(vals, updateFunc)
	require.NoError(t, err)
	require.True(t, changed)
	expected = []*int32{
		proto.Int32(-1), proto.Int32(1), proto.Int32(2),
		proto.Int32(3), proto.Int32(4), proto.Int32(5),
	}
	require.Equal(t, expected, newVals)

	// No changes
	vals = []*int32{
		proto.Int32(0), proto.Int32(1), proto.Int32(2),
		proto.Int32(3), proto.Int32(4), proto.Int32(5),
	}
	newVals, changed, err = updateAll(vals, updateFunc)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, vals, newVals)

	// Propagate error
	vals = []*int32{
		proto.Int32(0), proto.Int32(1), proto.Int32(2),
		proto.Int32(3), proto.Int32(-101), proto.Int32(5),
	}
	_, _, err = updateAll(vals, updateFunc)
	require.ErrorIs(t, err, errInvalid)
}
