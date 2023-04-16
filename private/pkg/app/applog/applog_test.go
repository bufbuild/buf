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

package applog

import (
    "testing"
	"fmt"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestGetZapLevel(t *testing.T) {
    testCases := []struct {
        levelString string
        expected    zapcore.Level
        expectError bool
    }{
        {"debug", zapcore.DebugLevel, false},
        {"info", zapcore.InfoLevel, false},
        {"warn", zapcore.WarnLevel, false},
        {"error", zapcore.ErrorLevel, false},
        {"", zapcore.InfoLevel, false},
        {"foobar", zapcore.InfoLevel, true},
    }

    for _, tc := range testCases {
        actual, err := getZapLevel(tc.levelString)
        if tc.expectError && err == nil {
            t.Errorf("Expected error for level %q but got none", tc.levelString)
        } else if !tc.expectError && err != nil {
            t.Errorf("Unexpected error for level %q: %s", tc.levelString, err)
        }
        if actual != tc.expected {
            t.Errorf("For level %q expected %v but got %v", tc.levelString, tc.expected, actual)
        }
    }
}

func TestGetZapEncoder(t *testing.T) {
	// Test valid formats
	testCases := []struct {
		format string
	}{
		{"text"},
		{"color"},
		{"json"},
		{"TEXT"},
		{"COLOR"},
		{"JSON"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("valid format %s", tc.format), func(t *testing.T) {
			encoder, err := getZapEncoder(tc.format)
			assert.NoError(t, err)
			assert.NotNil(t, encoder)
		})
	}

	// Test unknown format
	unknownFormat := "invalid"
	_, err := getZapEncoder(unknownFormat)
	assert.EqualError(t, err, fmt.Sprintf("unknown log format [text,color,json]: %q", unknownFormat))
}

