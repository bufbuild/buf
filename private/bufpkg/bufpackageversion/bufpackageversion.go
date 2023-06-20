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

package bufpackageversion

import "fmt"

func ShortDescription(name string) string {
	return fmt.Sprintf("Resolve module and %s plugin reference to a specific remote package version", name)
}

func LongDescription(registryName, commandName, examplePlugin string) string {
	return fmt.Sprintf(`This command returns the version of the %s asset to be used with the %s registry.
Examples:

Get the version of the eliza module and the %s plugin for use with %s.
    $ buf alpha package %s --module=buf.build/bufbuild/eliza --plugin=%s
        v1.7.0-20230609151053-e682db0d9918.1

Use a specific module version and plugin version.
    $ buf alpha package %s --module=buf.build/bufbuild/eliza:e682db0d99184be88b41c4405ea8a417 --plugin=%s:v1.0.0
        v1.0.0-20230609151053-e682db0d9918.1
`, registryName, registryName, examplePlugin, registryName, commandName, examplePlugin, commandName, examplePlugin)
}
