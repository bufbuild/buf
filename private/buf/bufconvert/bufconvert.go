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

package bufconvert

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufref"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const (
	// MessageEncodingBinpb is the binary image encoding.
	MessageEncodingBinpb MessageEncoding = iota + 1
	// MessageEncodingJSON is the JSON image encoding.
	MessageEncodingJSON
	// MessageEncodingTxtpb is the protobuf text image encoding.
	MessageEncodingTxtpb
	// MessageEncodingYAML is the YAML image encoding.
	MessageEncodingYAML
	// formatBinpb is the binary format.
	formatBinpb = "binpb"
	// formatJSON is the JSON format.
	formatJSON = "json"
	// formatTxtpb is the protobuf text format.
	formatTxtpb = "txtpb"
	// formatYAML is the YAML format.
	formatYAML = "yaml"

	// formatBin is the binary format's old form, now deprecated.
	formatBin = "bin"
)

var (
	// MessageEncodingFormatsString is the string representation of all message encoding formats.
	//
	// This does not include deprecated formats.
	MessageEncodingFormatsString = stringutil.SliceToString(messageEncodingFormats)
	// sorted
	messageEncodingFormats = []string{
		formatBinpb,
		formatJSON,
		formatTxtpb,
		formatYAML,
	}
)

// MessageEncoding is the encoding of the message
type MessageEncoding int

// MessageEncodingRef is a message encoding file reference.
type MessageEncodingRef interface {
	Path() string
	MessageEncoding() MessageEncoding
}

// NewMessageEncodingRef returns a new MessageEncodingRef.
func NewMessageEncodingRef(
	ctx context.Context,
	value string,
	defaultEncoding MessageEncoding,
) (MessageEncodingRef, error) {
	ctx, span := otel.GetTracerProvider().Tracer("bufbuild/buf").Start(ctx, "new_message_encoding_ref")
	defer span.End()
	path, messageEncoding, err := getPathAndMessageEncoding(ctx, value, defaultEncoding)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
	for key, value := range options {
		switch key {
		case "format":
			if app.IsDevNull(path) {
				return "", 0, fmt.Errorf("not allowed if path is %s", app.DevNullFilePath)
			}
			messageEncoding, err = parseMessageEncodingFormat(value)
			if err != nil {
				return "", 0, err
			}
		default:
			return "", 0, fmt.Errorf("invalid options key: %q", key)
		}
	}
	return path, messageEncoding, nil
}

func parseMessageEncodingExt(ext string, defaultEncoding MessageEncoding) MessageEncoding {
	switch strings.TrimPrefix(ext, ".") {
	case formatBin, formatBinpb:
		return MessageEncodingBinpb
	case formatJSON:
		return MessageEncodingJSON
	case formatTxtpb:
		return MessageEncodingTxtpb
	case formatYAML:
		return MessageEncodingYAML
	default:
		return defaultEncoding
	}
}

func parseMessageEncodingFormat(format string) (MessageEncoding, error) {
	switch format {
	case formatBin, formatBinpb:
		return MessageEncodingBinpb, nil
	case formatJSON:
		return MessageEncodingJSON, nil
	case formatTxtpb:
		return MessageEncodingTxtpb, nil
	case formatYAML:
		return MessageEncodingYAML, nil
	default:
		return 0, fmt.Errorf("invalid format for message: %q", format)
	}
}
