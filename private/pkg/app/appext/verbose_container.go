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

package appext

import (
	"github.com/bufbuild/buf/private/pkg/verbose"
)

type verboseContainer struct {
	verbosePrinter verbose.Printer
}

func newVerboseContainer(verbosePrinter verbose.Printer) *verboseContainer {
	return &verboseContainer{
		verbosePrinter: verbosePrinter,
	}
}

func (c *verboseContainer) VerboseEnabled() bool {
	return c.verbosePrinter.Enabled()
}

func (c *verboseContainer) VerbosePrinter() verbose.Printer {
	return c.verbosePrinter
}
