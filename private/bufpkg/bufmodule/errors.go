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

package bufmodule

import "errors"

// ErrNoTargetProtoFiles is the error to return if no target .proto files were found in situations where
// they were expected to be found.
//
// Pre-refactor, we had extremely exacting logic that determined if --path and --exclude-path were valid
// paths, which almost no CLI tool does. This logic had a heavy burden, when typically this error message
// is enough (and again, is more than almost any other CLI does - most CLIs silently move on if invalid
// paths are specified). The pre-refactor logic was the "allowNotExist" logic. Removing the allowNotExist
// logic was not a breaking change - we do not error in any place that we previously did not.
//
// This is used by bufctl.Controller.GetTargetImageWithConfigs, bufworkspace.NewWorkspaceForBucet, and bufimage.BuildImage.
//
// We do assume flag names here, but we're just going with reality.
var ErrNoTargetProtoFiles = errors.New("no .proto files were targeted. This can occur if no .proto files are found in your input, --path points to files that do not exist, or --exclude-path excludes all files.")
