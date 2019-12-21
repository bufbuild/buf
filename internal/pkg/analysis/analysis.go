// Package analysis implements an Annotation type to return annotations on files.
package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// Annotation is an annotation refencing a location within a file.
type Annotation struct {
	// Filename is the filename to display. This is typically a
	// relative path when produced by linters. If the filename
	// is not known, this will be empty.
	Filename string `json:"filename,omitempty" yaml:"filename,omitempty"`
	// StartLine is the starting line. If the line is not known,
	// this will be 0.
	StartLine int `json:"start_line,omitempty" yaml:"start_line,omitempty"`
	// StartColumn is the starting column. If the column is not
	// known, this will be 0.
	StartColumn int `json:"start_column,omitempty" yaml:"start_column,omitempty"`
	// EndLine is the ending line. If the line is not known,
	// this will be 0. This will be explicitly set to the
	// value of StartLine if the start and end line are the same.
	EndLine int `json:"end_line,omitempty" yaml:"end_line,omitempty"`
	// EndColumn is the ending column. If the column is not known,
	// this will be 0. This will be explicitly set to the
	// value of StartColumn if the start and end column are the same.
	EndColumn int `json:"end_column,omitempty" yaml:"end_column,omitempty"`
	// Type is the type of annotation, typically an ID representing
	// a lint failure type. This field is required.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// Message is the message of the annotation. This is required.
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

// String returns a basic string representation of a.
func (a *Annotation) String() string {
	filename := a.Filename
	line := a.StartLine
	column := a.StartColumn
	message := a.Message
	if filename == "" {
		filename = "<input>"
	}
	if line == 0 {
		line = 1
	}
	if column == 0 {
		column = 1
	}
	// should never happen but just in case
	if message == "" {
		message = a.Type
		if message == "" {
			message = "FAILURE"
		}
	}
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(filename)
	_, _ = buffer.WriteRune(':')
	_, _ = buffer.WriteString(strconv.Itoa(line))
	_, _ = buffer.WriteRune(':')
	_, _ = buffer.WriteString(strconv.Itoa(column))
	_, _ = buffer.WriteRune(':')
	_, _ = buffer.WriteString(message)
	return buffer.String()
}

// SortAnnotations sorts the Annotations.
//
// The order of sorting is:
//
//   Filename
//   StartLine
//   StartColumn
//   Type
//   Message
//   EndLine
//   EndColumn
func SortAnnotations(annotations []*Annotation) {
	sort.Stable(sortAnnotations(annotations))
}

type sortAnnotations []*Annotation

func (a sortAnnotations) Len() int          { return len(a) }
func (a sortAnnotations) Swap(i int, j int) { a[i], a[j] = a[j], a[i] }
func (a sortAnnotations) Less(i int, j int) bool {
	if a[i] == nil && a[j] == nil {
		return false
	}
	if a[i] == nil && a[j] != nil {
		return true
	}
	if a[i] != nil && a[j] == nil {
		return false
	}
	if a[i].Filename < a[j].Filename {
		return true
	}
	if a[i].Filename > a[j].Filename {
		return false
	}
	if a[i].StartLine < a[j].StartLine {
		return true
	}
	if a[i].StartLine > a[j].StartLine {
		return false
	}
	if a[i].StartColumn < a[j].StartColumn {
		return true
	}
	if a[i].StartColumn > a[j].StartColumn {
		return false
	}
	if a[i].Type < a[j].Type {
		return true
	}
	if a[i].Type > a[j].Type {
		return false
	}
	if a[i].Message < a[j].Message {
		return true
	}
	if a[i].Message > a[j].Message {
		return false
	}
	if a[i].EndLine < a[j].EndLine {
		return true
	}
	if a[i].EndLine > a[j].EndLine {
		return false
	}
	if a[i].EndColumn < a[j].EndColumn {
		return true
	}
	return false
}

// PrintAnnotations prints the annotations to the Writer.
//
// If asJSON is specified, the annotations are marshalled as JSON.
func PrintAnnotations(writer io.Writer, annotations []*Annotation, asJSON bool) error {
	if len(annotations) == 0 {
		return nil
	}
	for _, annotation := range annotations {
		s := ""
		if asJSON {
			data, err := json.Marshal(annotation)
			if err != nil {
				return err
			}
			s = string(data)
		} else {
			s = annotation.String()
		}
		if _, err := fmt.Fprintln(writer, s); err != nil {
			return err
		}
	}
	return nil
}
