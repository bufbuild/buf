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

package bufgenv2

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"google.golang.org/protobuf/types/descriptorpb"
)

// fieldOption is a field option
type fieldOption int

const (
	fieldOptionJsType fieldOption = iota + 1
)

var (
	allFieldOptions = []fieldOption{
		fieldOptionJsType,
	}
	fieldOptionToString = map[fieldOption]string{
		fieldOptionJsType: "jstype",
	}
	stringToFieldOption = map[string]fieldOption{
		"jstype": fieldOptionJsType,
	}
	fieldOptionToOverrideParseFunc = map[fieldOption]func(interface{}, fieldOption) (bufimagemodifyv2.Override, error){
		fieldOptionJsType: parseJSType,
	}
)

// String implements fmt.Stringer.
func (f fieldOption) String() string {
	s, ok := fieldOptionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// parseFieldOption parses the fieldOption.
//
// The empty string is an error.
func parseFieldOption(s string) (fieldOption, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, errors.New("empty field option")
	}
	f, ok := stringToFieldOption[s]
	if ok {
		return f, nil
	}
	return 0, fmt.Errorf("unknown field option: %q", s)
}

func parseJSType(override interface{}, fieldOption fieldOption) (bufimagemodifyv2.Override, error) {
	jsTypeName, ok := override.(string)
	if !ok {
		return nil, fmt.Errorf("invalid override for %v", fieldOption)
	}
	jsTypeEnum, ok := descriptorpb.FieldOptions_JSType_value[jsTypeName]
	if !ok {
		return nil, fmt.Errorf("%q is not a valid %v value, must be one of JS_NORMAL, JS_STRING and JS_NUMBER", jsTypeName, fieldOption)
	}
	jsType := descriptorpb.FieldOptions_JSType(jsTypeEnum)
	return bufimagemodifyv2.NewValueOverride(jsType), nil
}
