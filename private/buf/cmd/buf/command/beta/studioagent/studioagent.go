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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufstudioagent"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	// defaultShutdownTimeout is the default shutdown timeout.
	defaultShutdownTimeout = 10 * time.Second
	// defaultReadHeaderTimeout is the default read header timeout.
	defaultReadHeaderTimeout = 30 * time.Second
	// defaultIdleTimeout is the amount of time an HTTP/2 connection can be idle.
	defaultIdleTimeout = 3 * time.Minute
)

const (
	portFlagName              = "port"
	disallowedHeadersFlagName = "disallowed-header"
	forwardHeadersFlagName    = "forward-header"
	caCertFlagName            = "ca-cert"
	clientCertFlagName        = "client-cert"
	clientKeyFlagName         = "client-key"
	serverCertFlagName        = "server-cert"
	serverKeyFlagName         = "server-key"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <origin>",
		Short: "Run an HTTP(S) server as the studio agent with the origin be set as the allowed origin for CORS options.",
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
	CACert            string
	ClientCert        string
	ClientKey         string
	ServerCert        string
	ServerKey         string
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
		`The headers to be forwarded via the agent to the target server. Must be an equals sign separated key-value pair (like --forward-header=fromHeader1=toHeader1). Multiple header pairs are appended if specified multiple times.`,
	)
	flagSet.StringVar(
		&f.CACert,
		caCertFlagName,
		"",
		"The CA cert to be used in the client and server TLS configuration.",
	)
	flagSet.StringVar(
		&f.ClientCert,
		clientCertFlagName,
		"",
		"The cert to be used in the client TLS configuration.",
	)
	flagSet.StringVar(
		&f.ClientKey,
		clientKeyFlagName,
		"",
		"The key to be used in the client TLS configuration.",
	)
	flagSet.StringVar(
		&f.ServerCert,
		serverCertFlagName,
		"",
		"The cert to be used in the server TLS configuration.",
	)
	flagSet.StringVar(
		&f.ServerKey,
		serverKeyFlagName,
		"",
		"The key to be used in the server TLS configuration.",
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	// CA cert pool is optional. If it is nil, TLS uses the host's root CA set.
	var caCertPool *x509.CertPool
	var err error
	if flags.CACert != "" {
		caCertPool, err = newCACertPool(flags.CACert)
		if err != nil {
			return err
		}
	}
	// client TLS config is optional. If it is nil, it uses the default configuration from http2.Transport.
	var clientTLSConfig *tls.Config
	if flags.ClientCert != "" || flags.ClientKey != "" {
		clientTLSConfig, err = newTLSConfig(caCertPool, flags.ClientCert, flags.ClientKey)
		if err != nil {
			return fmt.Errorf("cannot create new client TLS config: %w", err)
		}
	}
	// server TLS config is optional. If it is nil, we serve with a h2c handler.
	var serverTLSConfig *tls.Config
	if flags.ServerCert != "" || flags.ServerKey != "" {
		serverTLSConfig, err = newTLSConfig(caCertPool, flags.ServerCert, flags.ServerKey)
		if err != nil {
			return fmt.Errorf("cannot create new server TLS config: %w", err)
		}
	}
	mux := bufstudioagent.NewHandler(
		container.Logger(),
		container.Arg(0), // the origin from command argument
		clientTLSConfig,
		stringutil.SliceToMap(flags.DisallowedHeaders),
		flags.ForwardHeaders,
	)
	httpServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		TLSConfig:         serverTLSConfig,
	}
	if serverTLSConfig == nil {
		httpServer.Handler = h2c.NewHandler(mux, &http2.Server{
			IdleTimeout: defaultIdleTimeout,
		})
	}
	var httpListenConfig net.ListenConfig
	httpListener, err := httpListenConfig.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%s", flags.Port))
	if err != nil {
		return err
	}
	jobs := []func(context.Context) error{
		func(ctx context.Context) error {
			return httpServe(httpServer, httpListener)
		},
		func(ctx context.Context) error {
			<-ctx.Done()
			ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
			defer cancel()
			return httpServer.Shutdown(ctx)
		},
	}
	if err := thread.Parallelize(ctx, jobs); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func newCACertPool(caCertFile string) (*x509.CertPool, error) {
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("error opening ca cert file %s", caCertFile)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	return caCertPool, nil
}

func newTLSConfig(caCertPool *x509.CertPool, certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("error creating x509 keypair from cert file %s and key file %s", certFile, keyFile)
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}, nil
}

func httpServe(httpServer *http.Server, listener net.Listener) error {
	if httpServer.TLSConfig != nil {
		return httpServer.ServeTLS(listener, "", "")
	}
	return httpServer.Serve(listener)
}
