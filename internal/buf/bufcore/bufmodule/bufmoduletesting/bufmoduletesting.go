// Copyright 2020 Buf Technologies, Inc.
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

const (
	//TestDigest is a valid testDigest.
	TestDigest = "b1-JR67l5Bfch-TiWnottksIFG9HyYS3o8JQKBTeZdGP-Y="
	// TestModuleNameString is a valid module name string.
	TestModuleNameString = "buf.build/foo/bar/v1"
)

// TestData is the data that maps to TestDigest with TestModuleNameString.
var TestData = map[string][]byte{
	"file1.proto":        []byte(`syntax="proto3";`),
	"folder/file2.proto": []byte(`syntax="proto3";`),
}
