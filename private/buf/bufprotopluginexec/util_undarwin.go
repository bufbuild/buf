// Copyright 2020-2024 Buf Technologies, Inc.
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

//go:build !darwin
// +build !darwin

package bufprotopluginexec

const tooManyFilesHelpMessage = `This is commonly caused by the maximum file limit being too low. Run "ulimit -n" to check your file limit. If this happened on generation, setting "strategy: all" for each configured plugin in your buf.gen.yaml can mitigate the issue if you are unable to change your file limit.`
