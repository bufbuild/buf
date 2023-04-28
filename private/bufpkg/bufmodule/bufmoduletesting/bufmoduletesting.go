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

package bufmoduletesting

import (
	breakingv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/breaking/v1"
	lintv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/lint/v1"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

const (
	// TestDigest is a valid digest.
	//
	// This matches TestData.
	TestDigest = "b1-gLO3B_5ClhdU52w1gMOxk4GokvCoM1OqjarxMfjStGQ="
	// TestDigestB3WithConfiguration is a valid digest.
	//
	// This matches TestDataWithConfiguration.
	TestDigestB3WithConfiguration = "b3-b2gkRgE1WxTKpEfsK4ql8STGxqc6nRimCeMBGB5i2OU="
	// TestDigestWithDocumentation is a valid test digest.
	//
	// This matches TestDataWithDocumentation.
	TestDigestWithDocumentation = "b1-Vqi49Lw-sr3tTLQVwSJrRJnJSwV0yeg97ea957z02B0="
	// TestDigestB3WithLicense is a valid digest.
	//
	// This matches TestDataWithLicense.
	TestDigestB3WithLicense = "b3-j7iu4iVzYQUFr97mbq2PNAlM5UjEnjtwEas0q7g4DVM="
	// TestModuleReferenceFooBarV1String is a valid module reference string.
	TestModuleReferenceFooBarV1String = "buf.build/foob/bar:v1"
	// TestModuleReferenceFooBarV2String is a valid module reference string.
	TestModuleReferenceFooBarV2String = "buf.build/foob/bar:v2"
	// TestModuleReferenceFooBazV1String is a valid module reference string.
	TestModuleReferenceFooBazV1String = "buf.build/foob/baz:v1"
	// TestModuleReferenceFooBazV2String is a valid module reference string.
	TestModuleReferenceFooBazV2String = "buf.build/foob/baz:v2"
	// TestModuleDocumentation is a markdown module documentation file.
	TestModuleDocumentation = "# Module Documentation"
	// TestModuleDocumentationPath is a path for module documentation file.
	TestModuleDocumentationPath = "README.md"
	// TestModuleLicense is a txt module license file.
	TestModuleLicense = "Module License"
	// TestModuleConfiguration is a configuration file with an arbitrary module name,
	// and example lint and breaking configuration that covers every key. At least two
	// items are included in every key (where applicable) so that we validate whether
	// or not the digest is deterministic.
	TestModuleConfiguration = `
version: v1
name: buf.build/acme/weather
lint:
  use:
    - DEFAULT
    - UNARY_RPC
  except:
    - BASIC
    - FILE_LOWER_SNAKE_CASE
  ignore:
    - file1.proto
    - folder/file2.proto
  ignore_only:
    ENUM_PASCAL_CASE:
      - file1.proto
      - folder
    BASIC:
      - file1.proto
      - folder
  enum_zero_value_suffix: _UNSPECIFIED
  rpc_allow_same_request_response: true
  rpc_allow_google_protobuf_empty_requests: true
  rpc_allow_google_protobuf_empty_responses: true
  service_suffix: Service
  allow_comment_ignores: true
breaking:
  use:
    - FILE
    - WIRE
  except:
    - FILE_NO_DELETE
    - RPC_NO_DELETE
  ignore:
    - file1.proto
    - folder/file2.proto
  ignore_only:
    FIELD_SAME_JSON_NAME:
      - file1.proto
      - folder
    WIRE:
      - file1.proto
      - folder
  ignore_unstable_packages: true
`
)

var (
	// TestData is the data that maps to TestDigest with TestModuleReferenceString.
	TestData = map[string][]byte{
		TestFile1Path: []byte(`syntax="proto3";`),
		TestFile2Path: []byte(`syntax="proto3";`),
	}
	//TestDataProto is the proto representation of TestData.
	TestDataProto = &modulev1alpha1.Module{
		Files: []*modulev1alpha1.ModuleFile{
			{
				Path:    TestFile1Path,
				Content: []byte(`syntax="proto3";`),
			},
			{
				Path:    TestFile2Path,
				Content: []byte(`syntax="proto3";`),
			},
		},
		BreakingConfig: &breakingv1.Config{Version: "v1beta1"},
		LintConfig:     &lintv1.Config{Version: "v1beta1"},
	}
	// TestDataWithDocumentation is the data that maps to TestDigestWithDocumentation.
	//
	// It includes a README.md file.
	TestDataWithDocumentation = map[string][]byte{
		TestFile1Path:               []byte(`syntax="proto3";`),
		TestModuleDocumentationPath: []byte(TestModuleDocumentation),
	}
	// TestDataWithDocumentationProto is the proto representation of TestDataWithDocumentation.
	TestDataWithDocumentationProto = &modulev1alpha1.Module{
		Files: []*modulev1alpha1.ModuleFile{
			{
				Path:    TestFile1Path,
				Content: []byte(`syntax="proto3";`),
			},
		},
		Documentation:     TestModuleDocumentation,
		DocumentationPath: TestModuleDocumentationPath,
		BreakingConfig:    &breakingv1.Config{Version: "v1beta1"},
		LintConfig:        &lintv1.Config{Version: "v1beta1"},
	}
	// TestDataWithConfiguration is the data that maps to TestDigestWithConfiguration.
	//
	// It includes a buf.yaml and a README.md file.
	TestDataWithConfiguration = map[string][]byte{
		TestFile1Path:               []byte(`syntax="proto3";`),
		TestFile2Path:               []byte(`syntax="proto3";`),
		"buf.yaml":                  []byte(TestModuleConfiguration),
		TestModuleDocumentationPath: []byte(TestModuleDocumentation),
	}
	// TestDataWithLicense is the data that maps to TestDigestB3WithLicense.
	//
	// It includes a LICENSE file.
	TestDataWithLicense = map[string][]byte{
		TestFile1Path: []byte(`syntax="proto3";`),
		"LICENSE":     []byte(TestModuleLicense),
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
