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
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// Handler is a Protobuf language server handler.
//
// For details, see https://github.com/golang/tools/tree/master/internal/lsp/protocol/typescript
type Handler interface {
	protocol.Server
}

// NewHandler returns a new Handler.
func NewHandler(logger *zap.Logger, engine Engine) Handler {
	return newHandler(
		logger,
		engine,
	)
}

// Engine is a Protobuf language server engine.
//
// This is used by both the Server that speaks the LSP,
// as well as the bufls sub-commands (e.g. 'bufls definition').
type Engine interface {
	Definition(context.Context, Location) (Location, error)
}

// NewEngine returns a new Protobuf language server engine.
func NewEngine(
	logger *zap.Logger,
	container appflag.Container,
	moduleConfigReader bufwire.ModuleConfigReader,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) Engine {
	return newEngine(
		logger,
		container,
		moduleConfigReader,
		moduleFileSetBuilder,
		imageBuilder,
	)
}

// Location is a source code location.
type Location interface {
	fmt.Stringer

	// Path is the unnormalized path of this location.
	Path() string
	// Line is the line number of the location.
	Line() int
	// Column is the column number of the location.
	Column() int
}

// ParseLocation parses a <filename>:<line>:<column> into a Location.
func ParseLocation(location string) (Location, error) {
	split := strings.Split(location, ":")
	if len(split) != 3 {
		return nil, fmt.Errorf("location %s is not structured as <filename>:<line>:<column>", location)
	}
	var (
		path   = split[0]
		line   = split[1]
		column = split[2]
	)
	lineNumber, err := strconv.Atoi(line)
	if err != nil {
		return nil, fmt.Errorf("location line %s must be an integer", line)
	}
	columnNumber, err := strconv.Atoi(column)
	if err != nil {
		return nil, fmt.Errorf("location column %s must be an integer", column)
	}
	return newLocation(
		path,
		lineNumber,
		columnNumber,
	)
}
