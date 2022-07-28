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

package bufls

import (
	"errors"
	"fmt"
	"net/url"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// textDocumentPositionParamsToLocation maps the protocol.TextDocumentPositionParams into
// a Location. The text document is zero-based, so we always make sure to increment the
// positional values by one.
func textDocumentPositionParamsToLocation(params protocol.TextDocumentPositionParams) (Location, error) {
	uri := params.TextDocument.URI
	if uri == "" {
		return nil, errors.New("text document uri is required")
	}
	path, err := uriToPath(uri)
	if err != nil {
		return nil, err
	}
	return newLocation(
		path,
		int(params.Position.Line)+1,
		int(params.Position.Character)+1,
	)
}

// uriToPath maps the URI into a filepath. We would normally be able to
// just call uri.Filename(), but the library panics if the URI is not a
// file:// scheme, so we may as well reimplement the lightweight transformation
// rather than wrapping this in a recover.
func uriToPath(protocolURI protocol.URI) (string, error) {
	parsedURI, err := url.ParseRequestURI(string(protocolURI))
	if err != nil {
		return "", fmt.Errorf("text document uri is invalid: %w", err)
	}
	if parsedURI.Scheme != uri.FileScheme {
		return "", fmt.Errorf("text document uri must specify the %s scheme, got %v", uri.FileScheme, parsedURI.Scheme)
	}
	return parsedURI.Path, nil
}

// locationToProtocolLocation maps the Location into a protocol.Location.
// The protocol.Location is zero-based, so we always decrement the positional
// values by one and validate that the final values are >= 0.
func locationToProtocolLocation(location Location) (protocol.Location, error) {
	protocolRange, err := locationToProtocolRange(location)
	if err != nil {
		return protocol.Location{}, err
	}
	return protocol.Location{
		URI:   uri.File(location.Path()),
		Range: protocolRange,
	}, nil
}

// locationToProtocolRange maps the Location into a protocol.Range.
// So far, we only need to capture the start of the range, so we leave
// the end position empty for now.
func locationToProtocolRange(location Location) (protocol.Range, error) {
	protocolPosition, err := locationToProtocolPosition(location)
	if err != nil {
		return protocol.Range{}, err
	}
	return protocol.Range{
		Start: protocolPosition,
	}, nil
}

// locationToProtocolPosition maps the Location into a protocol.Position.
func locationToProtocolPosition(location Location) (protocol.Position, error) {
	line := location.Line()
	if line <= 0 {
		return protocol.Position{}, fmt.Errorf("text document line must be >= 1, got %d", line)
	}
	column := location.Column()
	if column <= 0 {
		return protocol.Position{}, fmt.Errorf("text document column must be >= 1, got %d", column)
	}
	return protocol.Position{
		Line:      uint32(line) - 1,
		Character: uint32(column) - 1,
	}, nil
}
