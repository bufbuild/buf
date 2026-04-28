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

package protoencoding

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/standard/xos/xexec"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BenchmarkReparseExtensions_Synthetic exercises a hand-built FileDescriptorProto
// that mirrors the shape of a validate-annotated proto file with an unrecognized
// extension to force the reparse path.
func BenchmarkReparseExtensions_Synthetic(b *testing.B) {
	testFile, resolver := buildSyntheticInput(b)
	data, err := proto.Marshal(testFile)
	if err != nil {
		b.Fatal(err)
	}
	runReparseBench(b, resolver, data, func() proto.Message {
		return &descriptorpb.FileDescriptorProto{}
	})
}

// BenchmarkReparseExtensions_RealWorld builds a FileDescriptorSet from buf's
// own protos at benchmark startup and exercises ReparseExtensions on it.
func BenchmarkReparseExtensions_RealWorld(b *testing.B) {
	raw := buildBufRepoDescriptorSet(b)
	fdset := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(raw, fdset); err != nil {
		b.Fatal(err)
	}
	resolver, err := NewResolver(fdset.File...)
	if err != nil {
		b.Fatal(err)
	}
	runReparseBench(b, resolver, raw, func() proto.Message {
		return &descriptorpb.FileDescriptorSet{}
	})
}

func runReparseBench(
	b *testing.B,
	resolver Resolver,
	data []byte,
	newMessage func() proto.Message,
) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		fresh := newMessage()
		if err := proto.Unmarshal(data, fresh); err != nil {
			b.Fatal(err)
		}
		if err := ReparseExtensions(resolver, fresh.ProtoReflect()); err != nil {
			b.Fatal(err)
		}
	}
}

// buildBufRepoDescriptorSet invokes `go run ./cmd/buf build` against buf's
// own protos, writing the result to a tempdir, then returns the bytes.
func buildBufRepoDescriptorSet(b *testing.B) []byte {
	b.Helper()
	outputPath := filepath.Join(b.TempDir(), "buf_repo.binpb")
	var stderr bytes.Buffer
	err := xexec.Run(
		b.Context(),
		"go",
		xexec.WithArgs("run", "./cmd/buf", "build", "-o", outputPath),
		xexec.WithDir("../../.."), // relative to this package, this is the repo root.
		xexec.WithEnv(os.Environ()),
		xexec.WithStderr(&stderr),
	)
	if err != nil {
		b.Skipf("buf build failed: %v\n%s", err, stderr.String())
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		b.Fatal(err)
	}
	return raw
}

func buildSyntheticInput(b *testing.B) (*descriptorpb.FileDescriptorProto, Resolver) {
	b.Helper()

	descriptorFile := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)
	durationFile := protodesc.ToFileDescriptorProto(durationpb.File_google_protobuf_duration_proto)
	timestampFile := protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto)
	validateFile := protodesc.ToFileDescriptorProto(validate.File_buf_validate_validate_proto)

	const customOptionNum = 54321
	const customOptionVal = float32(3.14159)

	newFieldOpts := func() *descriptorpb.FieldOptions {
		fieldOpts := &descriptorpb.FieldOptions{}
		fieldRules := &validate.FieldRules{
			Required: proto.Bool(true),
			Type: &validate.FieldRules_Int32{
				Int32: &validate.Int32Rules{
					GreaterThan: &validate.Int32Rules_Gt{Gt: 0},
				},
			},
		}
		proto.SetExtension(fieldOpts, validate.E_Field, fieldRules)
		var unknownOption []byte
		unknownOption = protowire.AppendTag(unknownOption, customOptionNum, protowire.Fixed32Type)
		unknownOption = protowire.AppendFixed32(unknownOption, math.Float32bits(customOptionVal))
		fieldOpts.ProtoReflect().SetUnknown(unknownOption)
		return fieldOpts
	}

	const numMessages = 50
	const fieldsPerMessage = 20
	messages := make([]*descriptorpb.DescriptorProto, numMessages)
	for m := range messages {
		fields := make([]*descriptorpb.FieldDescriptorProto, fieldsPerMessage)
		for f := range fields {
			fields[f] = &descriptorpb.FieldDescriptorProto{
				Name:     proto.String(fmt.Sprintf("field_%d", f)),
				Number:   proto.Int32(int32(f + 1)),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				JsonName: proto.String(fmt.Sprintf("field_%d", f)),
				Options:  newFieldOpts(),
			}
		}
		messages[m] = &descriptorpb.DescriptorProto{
			Name:  proto.String(fmt.Sprintf("Msg%d", m)),
			Field: fields,
		}
	}
	testFile := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("test.proto"),
		Syntax:      proto.String("proto3"),
		Package:     proto.String("blah.blah"),
		Dependency:  []string{"buf/validate/validate.proto", "google/protobuf/descriptor.proto"},
		MessageType: messages,
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: proto.String(".google.protobuf.FieldOptions"),
				Name:     proto.String("baz"),
				Number:   proto.Int32(customOptionNum),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(),
			},
		},
	}
	resolver, err := NewResolver(descriptorFile, durationFile, timestampFile, validateFile, testFile)
	if err != nil {
		b.Fatal(err)
	}
	return testFile, resolver
}
