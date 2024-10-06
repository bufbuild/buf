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
	"fmt"
	"strconv"
	"strings"
)

const (
	// LogFormatText is the text log format.
	LogFormatText LogFormat = iota + 1
	// LogFormatColor is the colored text log format.
	//
	// This is the default value when parsing LogFormats. However, unless BuilderWithLoggerProvider
	// is used, there is no difference between LogFormatText and LogFormatColor.
	LogFormatColor
	// LogFormatJSON is the JSON log format.
	LogFormatJSON
)

// LogFormat is a format to print logs in.
type LogFormat int

// String implements fmt.Stringer
func (l LogFormat) String() string {
	switch l {
	case LogFormatText:
		return "text"
	case LogFormatColor:
		return "color"
	case LogFormatJSON:
		return "json"
	default:
		return strconv.Itoa(int(l))
	}
}

// ParseLogFormat parses the log format for the string.
//
// If logFormatString is empty, this returns LogFormatColor.
func ParseLogFormat(logFormatString string) (LogFormat, error) {
	logFormatString = strings.TrimSpace(strings.ToLower(logFormatString))
	switch logFormatString {
	case "text":
		return LogFormatText, nil
	case "color", "":
		return LogFormatColor, nil
	case "json":
		return LogFormatJSON, nil
	default:
		return 0, fmt.Errorf("unknown log format [text,color,json]: %q", logFormatString)
	}
}
