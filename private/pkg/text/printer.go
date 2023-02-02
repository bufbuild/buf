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
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	defaultIndent = "  "
)

var (
	newline            = []byte("\n")
	errNegativeIndents = errors.New("negative indents")
)

type printer struct {
	writer        io.Writer
	errorRecorder func(error)
	indent        string
	curIndents    int
}

func newPrinter(writer io.Writer, options ...PrinterOption) *printer {
	printer := &printer{
		writer: writer,
		indent: defaultIndent,
	}
	for _, option := range options {
		option(printer)
	}
	return printer
}

func (p *printer) P(args ...interface{}) {
	if len(args) == 0 {
		if _, err := p.writer.Write(newline); err != nil {
			p.recordError(err)
		}
		return
	}
	argBuffer := bytes.NewBuffer(nil)
	for _, arg := range args {
		if _, err := fmt.Fprint(argBuffer, arg); err != nil {
			p.recordError(err)
		}
	}
	if value := bytes.TrimSpace(argBuffer.Bytes()); len(value) > 0 {
		if p.curIndents > 0 {
			if _, err := p.writer.Write([]byte(strings.Repeat(p.indent, p.curIndents))); err != nil {
				p.recordError(err)
			}
		}
		if _, err := p.writer.Write(value); err != nil {
			p.recordError(err)
		}
	}
	if _, err := p.writer.Write(newline); err != nil {
		p.recordError(err)
	}
}

func (p *printer) In() {
	p.curIndents++
}

func (p *printer) Out() {
	if p.curIndents == 0 {
		p.recordError(errNegativeIndents)
	} else {
		p.curIndents--
	}
}

func (p *printer) recordError(err error) {
	if p.errorRecorder == nil {
		return
	}
	p.errorRecorder(err)
}
