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

// This package provides decoding and representations of git objects. Git has
// three kinds of objects: blob, tree, and commit. Blobs have no structure and
// hence have no decoding requirements. The other two are represented as
// [Tree], and [Commit] respectfully.
//
// Currently only Git repositories in SHA1 object format are supported.
package object
