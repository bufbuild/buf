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

import "io"

// Printer is a printer.
type Printer interface {
	// P writes the arguments and a newline with the current indent.
	//
	// Spaces are stripped from the end of the line, and only a newline is printed
	// if the args result in only spaces or no value.
	P(args ...interface{})
	// In indents.
	In()
	// Out unindents.
	Out()
}

// NewPrinter returns a new Printer.
func NewPrinter(writer io.Writer, options ...PrinterOption) Printer {
	return newPrinter(writer, options...)
}

// PrinterOption is an option for a printer.
type PrinterOption func(*printer)

// PrinterWithIndent returns a new PrinterOption that uses the given indent.
//
// The default is two spaces.
func PrinterWithIndent(indent string) PrinterOption {
	return func(printer *printer) {
		printer.indent = indent
	}
}

// PrinterWithErrorRecorder returns a new PrinterOption that records errors.
//
// The default is to drop errors.
func PrinterWithErrorRecorder(errorRecorder func(error)) PrinterOption {
	return func(printer *printer) {
		printer.errorRecorder = errorRecorder
	}
}
