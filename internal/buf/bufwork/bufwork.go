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

// Package bufwork defines the primitives used to enable workspaces.
//
// If a buf.work file exists in a parent directory (up to the root of
// the filesystem), the directory containing the file is used as the root of
// one or more modules. With this, modules can import from one another, and a
// variety of commands work on multiple modules rather than one. For example, if
// `buf lint` is run for an input that contains a buf.work, each of
// the modules contained within the workspace will be linted. Other commands, such
// as `buf build`, will merge workspace modules into one (i.e. a "supermodule")
// so that all of the files contained are consolidated into a single image.
//
// In the following example, the workspace consists of two modules: the module
// defined in the petapis directory can import definitions from the paymentapis
// module without vendoring the definitions under a common root. To be clear,
// `import "acme/payment/v2/payment.proto";` from the acme/pet/v1/pet.proto file
// will suffice as long as the buf.work file exists.
//
//   // buf.work
//   version: v1beta1
//   directories:
//     - paymentapis
//     - petapis
//
//   $ tree
//   .
//   ├── buf.work
//   ├── paymentapis
//   │   ├── acme
//   │   │   └── payment
//   │   │       └── v2
//   │   │           └── payment.proto
//   │   └── buf.yaml
//   └── petapis
//       ├── acme
//       │   └── pet
//       │       └── v1
//       │           └── pet.proto
//       └── buf.yaml
//
// Note that inputs MUST NOT overlap with any of the directories defined in the buf.work
// file. For example, it's not possible to build input "paymentapis/acme" since the image
// would otherwise include the content defined in paymentapis/acme/payment/v2/payment.proto as
// acme/payment/v2/payment.proto and payment/v2/payment.proto.
package bufwork

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// ExternalConfigV1Beta1FilePath is the default configuration file path for v1beta1.
const ExternalConfigV1Beta1FilePath = "buf.work"

// NewWorkspace returns a new workspace.
func NewWorkspace(
	ctx context.Context,
	config *Config,
	readBucket storage.ReadBucket,
	configProvider bufconfig.Provider,
	relativeRootPath string,
	targetSubDirPath string,
) (bufmodule.Workspace, error) {
	return newWorkspace(ctx, config, readBucket, configProvider, relativeRootPath, targetSubDirPath)
}

// Config is the workspace config.
type Config struct {
	// Directories are normalized and validated.
	Directories []string
}

// Provider provides workspace configurations.
type Provider interface {
	// GetConfig gets the Config for the YAML data at ConfigFilePath.
	//
	// If the data is of length 0, returns the default config.
	GetConfig(ctx context.Context, readBucket storage.ReadBucket, relativeRootPath string) (*Config, error)
	// GetConfig gets the Config for the given JSON or YAML data.
	//
	// If the data is of length 0, returns the default config.
	GetConfigForData(ctx context.Context, data []byte) (*Config, error)
}

// NewProvider returns a new Provider.
func NewProvider(logger *zap.Logger) Provider {
	return newProvider(logger)
}

type externalConfigV1Beta1 struct {
	Version     string   `json:"version,omitempty" yaml:"version,omitempty"`
	Directories []string `json:"directories,omitempty" yaml:"directories,omitempty"`
}

type externalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}
