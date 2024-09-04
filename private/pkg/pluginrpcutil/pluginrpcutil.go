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

package pluginrpcutil

import (
	"github.com/bufbuild/buf/private/pkg/command"
	"pluginrpc.com/pluginrpc"
)

// NewRunner returns a new pluginrpc.Runner for the command.Runner and program name.
func NewRunner(delegate command.Runner, programName string, options ...RunnerOption) pluginrpc.Runner {
	return newRunner(delegate, programName, options...)
}

// RunnerOption is an option for a new Runner.
type RunnerOption func(*runnerOptions)

// RunnerWithArgs returns a new RunnerOption that specifies a sub-command to invoke
// on the program.
func RunnerWithArgs(args ...string) RunnerOption {
	return func(runnerOptions *runnerOptions) {
		runnerOptions.args = args
	}
}
