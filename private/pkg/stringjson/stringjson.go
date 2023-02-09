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

package stringjson

import (
	"encoding/json"
	"fmt"
	"io"
)

var newline = []byte{'\n'}

// Object is an object that can either be marshaled to string or JSON.
type Object interface {
	fmt.Stringer
	json.Marshaler
}

// Print prints the object.
func Print(writer io.Writer, object Object, asJSON bool) error {
	return printInternal(writer, object, asJSON, nil)
}

// Println prints the object with a newline.
func Println(writer io.Writer, object Object, asJSON bool) error {
	return printInternal(writer, object, asJSON, newline)
}

func printInternal(writer io.Writer, object Object, asJSON bool, extra []byte) error {
	if object == nil {
		return nil
	}
	var data []byte
	var err error
	if asJSON {
		data, err = object.MarshalJSON()
		if err != nil {
			return err
		}
	} else {
		data = []byte(object.String())
	}
	if len(data) > 0 {
		if len(extra) > 0 {
			data = append(data, extra...)
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
	}
	return nil
}
