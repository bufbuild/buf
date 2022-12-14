// Copyright 2020-2022 Buf Technologies, Inc.
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

// Package buffeature provides known feature flags for the Buf CLI.
package buffeature

import "github.com/bufbuild/buf/private/pkg/app/appfeature"

// Available feature flags for the Buf CLI.
const (
	// TamperProofing enables the Buf CLI tamper proofing feature.
	// This enables sending of a manifest and blobs when pushing modules.
	// Additionally, it enables support for consuming a manifest and blobs when downloading modules.
	TamperProofing appfeature.FeatureFlag = "BUF_FEATURE_TAMPER_PROOFING"
)
