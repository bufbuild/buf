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

import "github.com/bufbuild/buf/internal/pkg/uuidutil"

const (
	// TestDigest is a valid digest.
	//
	// This matches TestData.
	TestDigest = "b1-gLO3B_5ClhdU52w1gMOxk4GokvCoM1OqjarxMfjStGQ="
	// TestModuleReferenceV1String is a valid module reference string.
	TestModuleReferenceFooBarV1String = "buf.build/foo/bar:v1"
	// TestModuleReferenceV2String is a valid module reference string.
	TestModuleReferenceFooBarV2String = "buf.build/foo/bar:v2"
	// TestModuleReferenceV1String is a valid module reference string.
	TestModuleReferenceFooBazV1String = "buf.build/foo/baz:v1"
	// TestModuleReferenceV2String is a valid module reference string.
	TestModuleReferenceFooBazV2String = "buf.build/foo/baz:v2"
)

var (
	// TestData is the data that maps to TestDigest with TestModuleReferenceString.
	TestData = map[string][]byte{
		"file1.proto":        []byte(`syntax="proto3";`),
		"folder/file2.proto": []byte(`syntax="proto3";`),
	}
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
	TestModuleReferenceFooBarCommitString = "buf.build/foo/bar:" + TestCommit
	TestModuleReferenceFooBazCommitString = "buf.build/foo/baz:" + TestCommit
}
