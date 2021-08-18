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

package main

import (
	"fmt"
	"os"
)

const deprecationMessage = `protoc-gen-buf-check-breaking has been moved to protoc-gen-buf-breaking.
Use protoc-gen-buf-breaking instead.

As one of the few changes buf will ever make, protoc-gen-buf-check-breaking was deprecated and
scheduled for removal for v1.0 in January 2021. In preparation for v1.0, instead of just printing
out a message notifying users of this, this command now returns an error for every invocation
and will be completely removed when v1.0 is released.

The only migration necessary is to change your installation and invocation
from protoc-gen-buf-check-breaking to protoc-gen-buf-breaking.
protoc-gen-buf-breaking can be installed in the exact same manner, whether
from GitHub Releases, Homebrew, AUR, or direct Go installation:

# instead of go get github.com/bufbuild/buf/cmd/protoc-gen-buf-check-breaking
go get github.com/bufbuild/buf/cmd/protoc-gen-buf-breaking
# instead of curl -sSL https://github.com/bufbuild/buf/releases/download/v0.51.1/protoc-gen-buf-check-breaking-Linux-x86_64
curl -sSL https://github.com/bufbuild/buf/releases/download/v0.51.1/protoc-gen-buf-breaking-Linux-x86_64

There is no change in functionality.`

func main() {
	fmt.Fprintln(os.Stderr, deprecationMessage)
	os.Exit(1)
}
