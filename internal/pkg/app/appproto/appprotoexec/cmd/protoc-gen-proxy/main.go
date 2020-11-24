// Copyright 2020 Buf Technologies, Inc.
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

// Package main defines protoc-gen-proxy, which is a testing protoc plugin that
// proxies to other plugins or to protoc.
//
// This should just be used for demonstration and testing purposes.
package main

import (
	"context"
	"errors"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"github.com/bufbuild/buf/internal/pkg/app/appproto/appprotoexec"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

// protoc --proxy_out=NAME:out foo.proto
// protoc --proxy_out=out --proxy_opt=NAME foo.proto
// protoc --proxy_out=NAME@param:out foo.proto
// protoc --proxy_out=out --proxy_opt=NAME@param foo.proto
// NAME@ must come at beginning

func main() {
	app.Main(context.Background(), appproto.NewRunFunc(appproto.HandlerFunc(handle)))
}

func handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseWriter,
	request *pluginpb.CodeGeneratorRequest,
) error {
	parameter := request.GetParameter()
	if parameter == "" {
		return errors.New("parameter empty")
	}
	split := strings.SplitN(parameter, "@", 2)
	logger, err := applog.NewLogger(container.Stderr(), "info", "text")
	if err != nil {
		return err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	handler, err := appprotoexec.NewHandler(logger, storageosProvider, split[0])
	if err != nil {
		return err
	}
	if len(split) == 2 {
		request.Parameter = proto.String(split[1])
	} else {
		request.Parameter = nil
	}
	return handler.Handle(ctx, container, responseWriter, request)
}
