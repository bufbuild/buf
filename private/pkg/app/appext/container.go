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
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"go.uber.org/zap"
)

type container struct {
	app.Container
	NameContainer
	LoggerContainer
	TracerContainer
	VerboseContainer
}

func newContainer(
	baseContainer app.Container,
	appName string,
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
) (*container, error) {
	nameContainer, err := newNameContainer(baseContainer, appName)
	if err != nil {
		return nil, err
	}
	return &container{
		Container:        baseContainer,
		NameContainer:    nameContainer,
		LoggerContainer:  newLoggerContainer(logger),
		TracerContainer:  newTracerContainer(appName),
		VerboseContainer: newVerboseContainer(verbosePrinter),
	}, nil
}
