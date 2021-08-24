// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufmoduletesting

import (
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

const (
	// TestDigest is a valid digest.
	//
	// This matches TestData.
	TestDigest = "b1-gLO3B_5ClhdU52w1gMOxk4GokvCoM1OqjarxMfjStGQ="
	// TestDigestWithDocumentation is a valid test digest.
	//
	// This matches TestDataWithDocumentation.
	TestDigestWithDocumentation = "b1-Vqi49Lw-sr3tTLQVwSJrRJnJSwV0yeg97ea957z02B0="
	// TestModuleReferenceFooBarV1String is a valid module reference string.
	TestModuleReferenceFooBarV1String = "buf.build/foob/bar:v1"
	// TestModuleReferenceFooBarV2String is a valid module reference string.
	TestModuleReferenceFooBarV2String = "buf.build/foob/bar:v2"
	// TestModuleReferenceFooBazV1String is a valid module reference string.
	TestModuleReferenceFooBazV1String = "buf.build/foob/baz:v1"
	// TestModuleReferenceFooBazV2String is a valid module reference string.
	TestModuleReferenceFooBazV2String = "buf.build/foob/baz:v2"
	// TestDocumentation is a markdown module documentation file.
	TestModuleDocumentation = "# Module Documentation"
)

var (
	// TestData is the data that maps to TestDigest with TestModuleReferenceString.
	TestData = map[string][]byte{
		TestFile1Path: []byte(`syntax="proto3";`),
		TestFile2Path: []byte(`syntax="proto3";`),
	}
	// TestDataWithDocumentation is the data that maps to TestDigestWithDocumentation.
	//
	// It includes a buf.md file.
	TestDataWithDocumentation = map[string][]byte{
		TestFile1Path: []byte(`syntax="proto3";`),
		"buf.md":      []byte(TestModuleDocumentation),
	}
	// TestFile1Path is the path of file1.proto.
	TestFile1Path = "file1.proto"
	// TestFile2Path is the path of file2.proto.
	TestFile2Path = "folder/file2.proto"
	// TestCommit is a valid commit.
	TestCommit string
	// TestModuleReferenceFooBarCommitString is a valid module reference string.
	TestModuleReferenceFooBarCommitString string
	// TestModuleReferenceFooBazCommitString is a valid module reference string.
	TestModuleReferenceFooBazCommitString string
)

func init() {
	testCommitUUID, err := uuidutil.New()
	if err != nil {
		panic(err.Error())
	}
	testCommitDashless, err := uuidutil.ToDashless(testCommitUUID)
	if err != nil {
		panic(err.Error())
	}
	TestCommit = testCommitDashless
	TestModuleReferenceFooBarCommitString = "buf.build/foob/bar:" + TestCommit
	TestModuleReferenceFooBazCommitString = "buf.build/foob/baz:" + TestCommit
}
