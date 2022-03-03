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

package bufwire

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/bufbuild/buf/private/buf/bufconvert"
	"github.com/bufbuild/buf/private/pkg/app"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type protoEncodingReader struct {
	logger *zap.Logger
}

var _ ProtoEncodingReader = &protoEncodingReader{}

func newProtoEncodingReader(
	logger *zap.Logger,
) *protoEncodingReader {
	return &protoEncodingReader{
		logger: logger,
	}
}

func (p *protoEncodingReader) GetMessagePayload(
	ctx context.Context,
	container app.EnvStdinContainer,
	messageRef bufconvert.MessageEncodingRef,
) (_ []byte, retErr error) {
	readCloser := io.NopCloser(container.Stdin())
	if messageRef.Path() != "-" {
		var err error
		readCloser, err = os.Open(messageRef.Path())
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	data, err := io.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("size of input message must not be zero")
	}
	return data, nil
}
