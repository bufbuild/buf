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

package studioagent

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufstudioagent"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/interrupt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	portFlagName              = "port"
	disallowedHeadersFlagName = "disallowed-header"
	forwardHeadersFlagName    = "forward-header"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <origin>",
		Short: "Run an HTTP server as the studio agent with the origin be set as the allowed origin for CORS options.",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Port              string
	DisallowedHeaders []string
	ForwardHeaders    map[string]string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Port,
		portFlagName,
		"8080",
		"The port to be exposed to accept HTTP requests.",
	)
	flagSet.StringSliceVar(
		&f.DisallowedHeaders,
		disallowedHeadersFlagName,
		nil,
		`The headers to be disallowed via the agent to the target server. Multiple headers are appended if specified multiple times.`,
	)
	flagSet.StringToStringVar(
		&f.ForwardHeaders,
		forwardHeadersFlagName,
		nil,
		`The headers to be forwarded via the agent to the target server. Must be a equal sign separated key-value pair (like --forward-header=fromHeader1=toHeader1). Multiple header pairs are appended if specified multiple times.`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	logger := container.Logger().Named("studio-agent")
	// convert the disallowedHeaders from a list of string to a map
	disallowedHeaders := make(map[string]struct{}, len(flags.DisallowedHeaders))
	for _, header := range flags.DisallowedHeaders {
		disallowedHeaders[header] = struct{}{}
	}
	config, err := bufcli.NewConfig(container)
	if err != nil {
		return err
	}
	mux := bufstudioagent.NewHandler(
		logger,
		container.Arg(0), // the origin from command argument
		config.TLS,
		disallowedHeaders,
		flags.ForwardHeaders,
	)
	server := http.Server{
		Addr:    ":" + flags.Port,
		Handler: requestLoggingHandler(mux, logger),
	}
	signalC, closer := interrupt.NewSignalChannel()
	go func() {
		logger.Info(fmt.Sprintf("listening on %s", server.Addr))
		if err = server.ListenAndServe(); err != nil {
			closer()
			return
		}
	}()
	<-signalC
	if err != nil {
		return err
	}
	if err := logger.Sync(); err != nil {
		return err
	}
	return server.Shutdown(ctx)
}

func requestLoggingHandler(mux http.Handler, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		bodyBytes, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, bufstudioagent.MaxMessageSizeBytesDefault))
		if err != nil {
			logger.Error("error when reading request body")
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		mux.ServeHTTP(w, r)
		logger.Info(
			"incoming request",
			zap.String("address", r.RemoteAddr),
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.Duration("duration", time.Since(start)),
			zap.ByteString("requestBody", bodyBytes),
		)
	})
}
