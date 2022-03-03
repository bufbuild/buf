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
	"os"

	"github.com/bufbuild/buf/private/buf/bufconvert"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/ioextended"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type protoEncodingWriter struct {
	logger *zap.Logger
}

var _ ProtoEncodingWriter = &protoEncodingWriter{}

func newProtoEncodingWriter(
	logger *zap.Logger,
) *protoEncodingWriter {
	return &protoEncodingWriter{
		logger: logger,
	}
}

func (p *protoEncodingWriter) PutMessagePayload(
	ctx context.Context,
	container app.EnvStdoutContainer,
	payload []byte,
	messageRef bufconvert.MessageEncodingRef,
) (retErr error) {
	writeCloser := ioextended.NopWriteCloser(container.Stdout())
	if messageRef.Path() != "-" {
		var err error
		writeCloser, err = os.Create(messageRef.Path())
		if err != nil {
			return err
		}
	}
	defer func() {
		retErr = multierr.Append(retErr, writeCloser.Close())
	}()
	_, err := writeCloser.Write(payload)
	return err
}
