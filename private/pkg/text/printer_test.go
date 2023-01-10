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

package text

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
)

func TestBasic(t *testing.T) {
	var printErr error
	buffer := bytes.NewBuffer(nil)
	p := NewPrinter(
		buffer,
		PrinterWithErrorRecorder(
			func(err error) {
				printErr = multierr.Append(printErr, err)
			},
		),
	)
	p.P()
	p.P("foo")
	p.In()
	p.P("1", "2")
	p.P("3", "4 ")
	p.Out()
	p.P(" ", " ")
	assert.Equal(t, buffer.String(), "\nfoo\n  12\n  34\n\n")
	assert.NoError(t, printErr)
	p.Out()
	assert.Equal(t, errNegativeIndents, printErr)
}
