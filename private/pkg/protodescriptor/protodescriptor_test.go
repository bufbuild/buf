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

package protodescriptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestValidateFileDescriptorEdition(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		edition descriptorpb.Edition
		wantErr bool
	}{
		{edition: descriptorpb.Edition_EDITION_PROTO3, wantErr: true},
		{edition: descriptorpb.Edition_EDITION_2023},
		{edition: descriptorpb.Edition_EDITION_2024},
		{edition: descriptorpb.Edition_EDITION_2026},
		{edition: descriptorpb.Edition_EDITION_99997_TEST_ONLY, wantErr: true},
	}
	for _, testcase := range testcases {
		t.Run(testcase.edition.String(), func(t *testing.T) {
			t.Parallel()
			err := ValidateFileDescriptor(&descriptorpb.FileDescriptorProto{
				Name:    new("a.proto"),
				Syntax:  new("editions"),
				Edition: testcase.edition.Enum(),
			})
			if testcase.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
