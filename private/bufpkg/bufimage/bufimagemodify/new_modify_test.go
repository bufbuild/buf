// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufimagemodify

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModify(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description           string
		moduleNameToDirpath   string
		config                bufconfig.Config
		pathToExpectedOptions map[string]*descriptorpb.FileOptions
	}{}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
		})
	}
}

func TestSweepFileOption(t *testing.T) {
	t.Parallel()
	// TODO
}

func TestSweepFieldOption(t *testing.T) {
	t.Parallel()
	// TODO in v2
}
