// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufconvert

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufref"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.opencensus.io/trace"
)

const (
	// MessageEncodingBin is the binary image encoding.
	MessageEncodingBin MessageEncoding = iota + 1
	// MessageEncodingJSON is the JSON image encoding.
	MessageEncodingJSON
	// formatBin is the binary format.
	formatBin = "bin"
	// formatJSON is the JSON format.
	formatJSON = "json"
)

var (
	// MessageEncodingFormatsString is the string representation of all message encoding formats.
	//
	// This does not include deprecated formats.
	MessageEncodingFormatsString = stringutil.SliceToString(messageEncodingFormats)
	// sorted
	messageEncodingFormats = []string{
		formatBin,
		formatJSON,
	}
)

// MessageEncoding is the encoding of the message
type MessageEncoding int

type MessageEncodingRef interface {
	Path() string
	MessageEncoding() MessageEncoding
}

func NewMessageEncodingRef(
	ctx context.Context,
	value string,
	defaultEncoding MessageEncoding,
) (MessageEncodingRef, error) {
	ctx, span := trace.StartSpan(ctx, "new_message_encoding_ref")
	defer span.End()
	path, messageEncoding, err := getPathAndMessageEncoding(ctx, value, defaultEncoding)
	if err != nil {
		return nil, err
	}
	return newMessageEncodingRef(path, messageEncoding), nil
}

func getPathAndMessageEncoding(
	ctx context.Context,
	value string,
	defaultEncoding MessageEncoding,
) (string, MessageEncoding, error) {
	path, options, err := bufref.GetRawPathAndOptions(value)
	if err != nil {
		return "", 0, err
	}
	messageEncoding := parseMessageEncodingExt(filepath.Ext(path), defaultEncoding)
	if format, ok := options["format"]; ok {
		messageEncoding, err = parseMessageEncodingFormat(format)
		if err != nil {
			return "", 0, err
		}
	}
	return path, messageEncoding, nil
}

func parseMessageEncodingExt(ext string, defaultEncoding MessageEncoding) MessageEncoding {
	switch strings.TrimPrefix(ext, ".") {
	case formatBin:
		return MessageEncodingBin
	case formatJSON:
		return MessageEncodingJSON
	default:
		return defaultEncoding
	}
}

func parseMessageEncodingFormat(format string) (MessageEncoding, error) {
	switch format {
	case formatBin:
		return MessageEncodingBin, nil
	case formatJSON:
		return MessageEncodingJSON, nil
	default:
		return 0, fmt.Errorf("invalid format for message: %q", format)
	}
}
