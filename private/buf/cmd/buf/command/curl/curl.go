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

package curl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufcurl"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
	"golang.org/x/net/http2"
)

const (
	// Input schema flags
	schemaFlagName = "schema"

	// Reflection flags
	reflectFlagName         = "reflect"
	reflectHeaderFlagName   = "reflect-header"
	reflectProtocolFlagName = "reflect-protocol"

	// Protocol/transport flags
	protocolFlagName            = "protocol"
	unixSocketFlagName          = "unix-socket"
	http2PriorKnowledgeFlagName = "http2-prior-knowledge"

	// TLS flags
	keyFlagName        = "key"
	certFlagName       = "cert"
	caCertFlagName     = "cacert"
	serverNameFlagName = "servername"
	insecureFlagName   = "insecure"

	// Timeout flags
	noKeepAliveFlagName    = "no-keepalive"
	keepAliveFlagName      = "keepalive-time"
	connectTimeoutFlagName = "connect-timeout"

	// Header and request body flags
	userAgentFlagName = "user-agent"
	headerFlagName    = "header"
	dataFlagName      = "data"
	outputFlagName    = "output"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <url>",
		Short: "Invoke an RPC endpoint, a la 'cURL'",
		Long: `This command helps you invoke HTTP RPC endpoints on a server that uses gRPC or Connect.

By default, server reflection is used, unless the --reflect flag is set to false. Without server
reflection, a --schema flag must be provided to indicate the Protobuf schema for the method being
invoked.

The only positional argument is the URL of the RPC method to invoke. The name of the method to
invoke comes from the last two path components of the URL, which should be the fully-qualified
service name and method name, respectively.

The URL can use either http or https as the scheme. If http is used then HTTP 1.1 will be used
unless the --http2-prior-knowledge flag is set. If https is used then HTTP/2 will be preferred
during protocol negotiation and HTTP 1.1 used only if the server does not support HTTP/2.

The default RPC protocol used will be Connect. To use a different protocol (gRPC or gRPC-Web),
use the --protocol flag. Note that the gRPC protocol cannot be used with HTTP 1.1.

The input request is specified via the -d or --data flag. If absent, an empty request is sent. If
the flag value starts with an at-sign (@), then the rest of the flag value is interpreted as a
filename from which to read the request body. If that filename is just a dash (-), then the request
body is read from stdin. The request body is a JSON document that contains the JSON formatted
request message. If the RPC method being invoked is a client-streaming method, the request body may
consist of multiple JSON values, appended to one another. Multiple JSON documents should usually be
separated by whitespace, though this is not strictly required unless the request message type has a
custom JSON representation that is not a JSON object.

Request metadata (i.e. headers) are defined using -H or --header flags. The flag value is in
"name: value" format. But if it starts with an at-sign (@), the rest of the value is interpreted as
a filename from which headers are read, each on a separate line. If the filename is just a dash (-),
then the headers are read from stdin.

If headers and the request body are both to be read from the same file (or both read from stdin),
the file must include headers first, then a blank line, and then the request body.

Examples:

Issue a unary RPC to a plain-text (i.e. "h2c") gRPC server, where the schema for the service is
in a Buf module in the current directory, using an empty request message:

    $ buf curl --schema . --protocol grpc --http2-prior-knowledge  \
         http://localhost:20202/foo.bar.v1.FooService/DoSomething

Issue an RPC to a Connect server, where the schema comes from the Buf Schema Registry, using
a request that is defined as a command-line argument:

    $ buf curl --schema buf.build/bufbuild/eliza  \
         --data '{"name": "Bob Loblaw"}'          \
         https://demo.connect.build/buf.connect.demo.eliza.v1.ElizaService/Introduce

Issue a unary RPC to a server that supports reflection, with verbose output:

    $ buf curl --data '{"sentence": "I am not feeling well."}' -v  \
         https://demo.connect.build/buf.connect.demo.eliza.v1.ElizaService/Say

Issue a client-streaming RPC to a gRPC-web server that supports reflection, where custom
headers and request data are both in a heredoc:

    $ buf curl --data @- --header @- --protocol grpcweb                              \
         https://demo.connect.build/buf.connect.demo.eliza.v1.ElizaService/Converse  \
       <<EOM
    Custom-Header-1: foo-bar-baz
    Authorization: token jas8374hgnkvje9wpkerebncjqol4

    {"sentence": "Hi, doc. I feel hungry."}
    {"sentence": "What is the answer to life, the universe, and everything?"}
    {"sentence": "If you were a fish, what of fish would you be?."}
    EOM

Note that server reflection (i.e. use of the --reflect flag) does not work with HTTP 1.1 since the
protocol relies on bidirectional streaming. If server reflection is used, the assumed URL for the
reflection service is the same as the given URL, but with the last two elements removed and
replaced with the service and method name for server reflection.

If an error occurs that is due to incorrect usage or other unexpected error, this program will
return an exit code that is less than 8. If the RPC fails otherwise, this program will return an
exit code that is the gRPC code, shifted three bits to the left.
`,
		Args: checkPositionalArgs,
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
	// Flags for defining input schema
	Schema string

	// Flags for server reflection
	Reflect         bool
	ReflectHeaders  []string
	ReflectProtocol string

	// Protocol details
	Protocol            string
	UnixSocket          string
	HTTP2PriorKnowledge bool

	// TLS
	Key, Cert, CACert, ServerName string
	Insecure                      bool
	// TODO: CRLFile, CertStatus

	// Timeouts
	NoKeepAlive           bool
	KeepAliveTimeSeconds  float64
	ConnectTimeoutSeconds float64

	// Handling request and response data and metadata
	UserAgent string
	Headers   []string
	Data      string
	Output    string

	// so we can inquire about which flags present on command-line
	// TODO: ideally we'd use cobra directly instead of having the appcmd wrapper,
	//  which prevents a lot of basic functionality by not exposing many cobra features
	flagSet *pflag.FlagSet
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	f.flagSet = flagSet

	flagSet.StringVar(
		&f.Schema,
		schemaFlagName,
		"",
		fmt.Sprintf(
			`The module to use for the RPC schema. This is necessary if the server does not support
server reflection. The format of this argument is the same as for the <input> arguments to
other buf sub-commands such as build and generate. It can indicate a directory, a file, a
remote module in the Buf Schema Registry, or even standard in ("-") for feeding an image or
file descriptor set to the command in a shell pipeline.
Setting this flags implies --%s=false`,
			reflectFlagName,
		),
	)
	flagSet.BoolVar(
		&f.Reflect,
		reflectFlagName,
		true,
		`If true, use server reflection to determine the schema`,
	)
	flagSet.StringSliceVar(
		&f.ReflectHeaders,
		reflectHeaderFlagName,
		nil,
		`Request headers to include with reflection requests. This flag may only be used
when --reflect is also set. This flag may be specified more than once to indicate
multiple headers. Each flag value should have the form "name: value". But a special value
of '*' may be used to indicate that all normal request headers (from --header and -H
flags) should also be included with reflection requests. A special value of '@<path>'
means to read headers from the file at <path>. If the path is "-" then headers are
read from stdin. It is not allowed to indicate a file with the same path as used with
the request data flag (--data or -d). Furthermore, it is not allowed to indicate stdin
if the schema is expected to be provided via stdin as a file descriptor set or image`,
	)
	flagSet.StringVar(
		&f.ReflectProtocol,
		reflectProtocolFlagName,
		"",
		`The reflection protocol to use for downloading information from the server. This flag
may only be used when server reflection is used. By default, this command will try all known
reflection protocols from newest to oldest. If this results in a "Not Implemented" error,
then older protocols will be used. In practice, this means that "grpc-v1" is tried first,
and "grpc-v1alpha" is used if it doesn't work. If newer reflection protocols are introduced,
they may be preferred in the absence of this flag being explicitly set to a specific protocol.
The valid values for this flag are "grpc-v1" and "grpc-v1alpha". These correspond to services
named "grpc.reflection.v1.ServerReflection" and "grpc.reflection.v1alpha.ServerReflection"
respectively`,
	)

	flagSet.StringVar(
		&f.Protocol,
		protocolFlagName,
		connect.ProtocolConnect,
		`The RPC protocol to use. This can be one of "grpc", "grpcweb", or "connect"`,
	)
	flagSet.StringVar(
		&f.UnixSocket,
		unixSocketFlagName,
		"",
		`The path to a unix socket that will be used instead of opening a TCP socket to the host
and port indicated in the URL`,
	)
	flagSet.BoolVar(
		&f.HTTP2PriorKnowledge,
		http2PriorKnowledgeFlagName,
		false,
		`This flag can be used with URLs that use the http scheme (as opposed to https) to indicate
that HTTP/2 should be used. Without this, HTTP 1.1 will be used with URLs with an http
scheme. For https scheme, HTTP/2 will be negotiate during the TLS handshake if the server
supports it (otherwise HTTP 1.1 is used)`,
	)

	flagSet.BoolVar(
		&f.NoKeepAlive,
		noKeepAliveFlagName,
		false,
		`By default, connections are created using TCP keepalive. If this flag is present, they
will be disabled`,
	)
	flagSet.Float64Var(
		&f.KeepAliveTimeSeconds,
		keepAliveFlagName,
		60,
		`The duration, in seconds, between TCP keepalive transmissions`,
	)
	flagSet.Float64Var(
		&f.ConnectTimeoutSeconds,
		connectTimeoutFlagName,
		0,
		`The time limit, in seconds, for a connection to be established with the server. There is
no limit if this flag is not present`,
	)

	flagSet.StringVar(
		&f.Key,
		keyFlagName,
		"",
		`Path to a PEM-encoded X509 private key file, for using client certificates with TLS. This
option is only valid when the URL uses the https scheme. A --cert flag must also be
present to provide tha certificate and public key that corresponds to the given
private key`,
	)
	flagSet.StringVarP(
		&f.Cert,
		certFlagName,
		"E",
		"",
		`Path to a PEM-encoded X509 certificate file, for using client certificates with TLS. This
option is only valid when the URL uses the https scheme. A --key flag must also be
present to provide tha private key that corresponds to the given certificate`,
	)
	flagSet.StringVar(
		&f.CACert,
		caCertFlagName,
		"",
		`Path to a PEM-encoded X509 certificate pool file that contains the set of trusted
certificate authorities/issuers. If omitted, the system's default set of trusted
certificates are used to verify the server's certificate. This option is only valid
when the URL uses the https scheme. It is not applicable if --insecure flag is used`,
	)
	flagSet.BoolVarP(
		&f.Insecure,
		insecureFlagName,
		"k",
		false,
		`If set, the TLS connection will be insecure and the server's certificate will NOT be
verified. This is generally discouraged. This option is only valid when the URL uses
the https scheme`,
	)
	flagSet.StringVar(
		&f.ServerName,
		serverNameFlagName,
		"",
		`The server name to use in TLS handshakes (for SNI) if the URL scheme is https. If not
specified, the default is the origin host in the URL or the value in a "Host" header if
one is provided`,
	)

	flagSet.StringVarP(
		&f.UserAgent,
		userAgentFlagName,
		"A",
		"",
		`The user agent string to send`,
	)
	flagSet.StringSliceVarP(
		&f.Headers,
		headerFlagName,
		"H",
		nil,
		`Request headers to include with the RPC invocation. This flag may be specified more
than once to indicate multiple headers. Each flag value should have the form "name: value".
A special value of '@<path>' means to read headers from the file at <path>. If the path
is "-" then headers are read from stdin. If the same file is indicated as used with the
request data flag (--data or -d), the file must contain all headers, then a blank line,
and then the request body. It is not allowed to indicate stdin if the schema is expected
to be provided via stdin as a file descriptor set or image`,
	)
	flagSet.StringVarP(
		&f.Data,
		dataFlagName,
		"d",
		"",
		`Request data. This should be zero or more JSON documents, each indicating a request
message. For unary RPCs, there should be exactly one JSON document. A special value of
'@<path>' means to read the data from the file at <path>. If the path is "-" then the
request data is read from stdin. If the same file is indicated as used with the request
headers flags (--header or -H), the file must contain all headers, then a blank line, and
then the request body. It is not allowed to indicate stdin if the schema is expected to be
provided via stdin as a file descriptor set or image`,
	)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		"o",
		"",
		`Path to output file to create with response data. If absent, response is printed to stdout`,
	)
}

func (f *flags) validate(isSecure bool) error {
	if (f.Key != "" || f.Cert != "" || f.CACert != "" || f.ServerName != "" || f.flagSet.Changed(insecureFlagName)) &&
		!isSecure {
		return fmt.Errorf(
			"TLS flags (--%s, --%s, --%s, --%s, --%s) should not be used unless URL is secure (https)",
			keyFlagName, certFlagName, caCertFlagName, insecureFlagName, serverNameFlagName)
	}
	if (f.Key != "") != (f.Cert != "") {
		return fmt.Errorf("if one of --%s or --%s flags is used, both should be used (mutual TLS with a client certificate requires both)", keyFlagName, certFlagName)
	}
	if f.Insecure && f.CACert != "" {
		return fmt.Errorf("if --%s is set, --%s should not be set as it is unused", insecureFlagName, caCertFlagName)
	}

	if f.HTTP2PriorKnowledge && isSecure {
		return fmt.Errorf("--%s flag is not for use with secure URLs (https) since http/2 can be negotiated during TLS handshake", http2PriorKnowledgeFlagName)
	}
	if !isSecure && !f.HTTP2PriorKnowledge && f.Protocol == connect.ProtocolGRPC {
		return fmt.Errorf("grpc protocol cannot be used with plain-text URLs (http) unless --%s flag is set", http2PriorKnowledgeFlagName)
	}

	if (len(f.ReflectHeaders) > 0 || f.flagSet.Changed(reflectProtocolFlagName)) && !f.Reflect {
		return fmt.Errorf(
			"reflection flags (--%s, --%s) should not be used if --%s is false",
			reflectHeaderFlagName, reflectProtocolFlagName, reflectFlagName)
	}
	if _, err := bufcurl.ParseReflectProtocol(f.ReflectProtocol); err != nil {
		return fmt.Errorf(
			"--%s value must be one of %s",
			reflectProtocolFlagName,
			stringutil.SliceToHumanStringOrQuoted(bufcurl.AllKnownReflectProtocolStrings),
		)
	}
	switch f.Protocol {
	case connect.ProtocolConnect, connect.ProtocolGRPC, connect.ProtocolGRPCWeb:
	default:
		return fmt.Errorf(
			"--%s value must be one of %q, %q, or %q",
			protocolFlagName, connect.ProtocolConnect, connect.ProtocolGRPC, connect.ProtocolGRPCWeb)
	}

	if f.NoKeepAlive && f.flagSet.Changed(keepAliveFlagName) {
		return fmt.Errorf("--%s should not be specified if keepalive is disabled", keepAliveFlagName)
	}
	if f.KeepAliveTimeSeconds <= 0 {
		return fmt.Errorf("--%s value must be positive", keepAliveFlagName)
	}
	// these two default to zero (which means no timeout in effect)
	if f.ConnectTimeoutSeconds < 0 || (f.ConnectTimeoutSeconds == 0 && f.flagSet.Changed(connectTimeoutFlagName)) {
		return fmt.Errorf("--%s value must be positive", connectTimeoutFlagName)
	}

	if f.Schema != "" && f.Reflect {
		if f.flagSet.Changed(reflectFlagName) {
			// explicitly enabled both
			return fmt.Errorf("cannot specify both --%s and --%s", schemaFlagName, reflectFlagName)
		}
		// Reflect just has default value; unset it since we're going to use --schema instead
		f.Reflect = false
	}
	if !f.Reflect && f.Schema == "" {
		return fmt.Errorf("must specify --%s if --%s is false", schemaFlagName, reflectFlagName)
	}
	schemaIsStdin := strings.HasPrefix(f.Schema, "-")

	var dataFile string
	if strings.HasPrefix(f.Data, "@") {
		dataFile = strings.TrimPrefix(f.Data, "@")
		if dataFile == "" {
			return fmt.Errorf("--%s value starting with '@' must indicate '-' for stdin or a filename", dataFlagName)
		}
		if dataFile == "-" && schemaIsStdin {
			return fmt.Errorf("--%s and --%s flags cannot both indicate reading from stdin", schemaFlagName, dataFlagName)
		}
	}

	headerFiles := map[string]struct{}{}
	if err := validateHeaders(f.Headers, headerFlagName, schemaIsStdin, false, headerFiles); err != nil {
		return err
	}
	reflectHeaderFiles := map[string]struct{}{}
	if err := validateHeaders(f.ReflectHeaders, reflectHeaderFlagName, schemaIsStdin, true, reflectHeaderFiles); err != nil {
		return err
	}
	for file := range reflectHeaderFiles {
		if file == dataFile {
			return fmt.Errorf("--%s and --%s flags cannot indicate the same source", dataFlagName, reflectHeaderFlagName)
		}
	}

	return nil
}

func validateHeaders(flags []string, flagName string, schemaIsStdin bool, allowAsterisk bool, headerFiles map[string]struct{}) error {
	var hasAsterisk bool
	for _, header := range flags {
		switch {
		case strings.HasPrefix(header, "@"):
			file := strings.TrimPrefix(header, "@")
			if _, ok := headerFiles[file]; ok {
				return fmt.Errorf("multiple --%s values refer to the same file %s", flagName, file)
			}
			if file == "" {
				return fmt.Errorf("--%s value starting with '@' must indicate '-' for stdin or a filename", flagName)
			}
			if file == "-" && schemaIsStdin {
				return fmt.Errorf("--%s and --%s flags cannot both indicate reading from stdin", schemaFlagName, flagName)
			}
			headerFiles[file] = struct{}{}
		case header == "*":
			if !allowAsterisk {
				return fmt.Errorf("--%s value '*' is not valid", flagName)
			}
			if hasAsterisk {
				return fmt.Errorf("multiple --%s values both indicate '*'", flagName)
			}
			hasAsterisk = true
		case header == "":
			return fmt.Errorf("--%s value cannot be blank", flagName)
		case strings.ContainsRune(header, '\n'):
			return fmt.Errorf("--%s value cannot contain a newline", flagName)
		default:
			parts := strings.SplitN(header, ":", 2)
			if len(parts) < 2 {
				return fmt.Errorf("--%s value is a malformed header: %q", flagName, header)
			}
		}
	}

	return nil
}

func verifyEndpointURL(urlArg string) (endpointURL *url.URL, service, method, baseURL string, err error) {
	endpointURL, err = url.Parse(urlArg)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("%q is not a valid endpoint URL: %w", urlArg, err)
	}
	if endpointURL.Scheme != "http" && endpointURL.Scheme != "https" {
		return nil, "", "", "", fmt.Errorf("invalid endpoint URL: sceme %q is not supported", endpointURL.Scheme)
	}

	if strings.HasSuffix(endpointURL.Path, "/") {
		return nil, "", "", "", fmt.Errorf("invalid endpoint URL: path %q should not end with a slash (/)", endpointURL.Path)
	}
	parts := strings.Split(endpointURL.Path, "/")
	if len(parts) < 2 || parts[len(parts)-1] == "" || parts[len(parts)-2] == "" {
		return nil, "", "", "", fmt.Errorf("invalid endpoint URL: path %q should end with two non-empty components indicating service and method", endpointURL.Path)
	}
	service, method = parts[len(parts)-2], parts[len(parts)-1]
	baseURL = strings.TrimSuffix(urlArg, service+"/"+method)
	if baseURL == urlArg {
		// should not be possible due to above checks
		return nil, "", "", "", fmt.Errorf("failed to extract base URL from %q", urlArg)
	}
	return endpointURL, service, method, baseURL, nil
}

func checkPositionalArgs(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("expecting exactly one positional argument: the URL of the endpoint to invoke")
	}
	_, _, _, _, err := verifyEndpointURL(args[0])
	return err
}

func run(ctx context.Context, container appflag.Container, f *flags) (err error) {
	endpointURL, service, method, baseURL, err := verifyEndpointURL(container.Arg(0))
	if err != nil {
		return err
	}
	isSecure := endpointURL.Scheme == "https"
	if err := f.validate(isSecure); err != nil {
		return err
	}

	var clientOptions []connect.ClientOption
	switch f.Protocol {
	case connect.ProtocolGRPC:
		clientOptions = []connect.ClientOption{connect.WithGRPC()}
	case connect.ProtocolGRPCWeb:
		clientOptions = []connect.ClientOption{connect.WithGRPCWeb()}
	}
	if f.Protocol != connect.ProtocolGRPC {
		// The transport will log trailers to the verbose printer. But if
		// we're not using standard grpc protocol, trailers are actually encoded
		// in an end-of-stream message for streaming calls. So this interceptor
		// will print the trailers for streaming calls when the response stream
		// is drained.
		clientOptions = append(clientOptions, connect.WithInterceptors(bufcurl.TraceTrailersInterceptor(container.VerbosePrinter())))
	}

	dataSource := "(argument)"
	var dataFileReference string
	if strings.HasPrefix(f.Data, "@") {
		dataFileReference = strings.TrimPrefix(f.Data, "@")
		if dataFileReference == "-" {
			dataSource = "(stdin)"
		} else {
			dataSource = dataFileReference
			if absFile, err := filepath.Abs(dataFileReference); err == nil {
				dataFileReference = absFile
			}
		}
	}
	requestHeaders, dataReader, err := bufcurl.LoadHeaders(f.Headers, dataFileReference, nil)
	if err != nil {
		return err
	}
	if len(requestHeaders.Values("user-agent")) == 0 {
		userAgent := f.UserAgent
		if userAgent == "" {
			userAgent = bufcurl.DefaultUserAgent(f.Protocol, bufcli.Version)
		}
		requestHeaders.Set("user-agent", userAgent)
	}
	if dataReader == nil {
		if dataFileReference == "-" {
			dataReader = os.Stdin
		} else if dataFileReference != "" {
			f, err := os.Open(dataFileReference)
			if err != nil {
				return bufcurl.ErrorHasFilename(err, dataFileReference)
			}
			dataReader = f
		} else if f.Data != "" {
			dataReader = io.NopCloser(strings.NewReader(f.Data))
		}
		// dataReader is left nil when nothing specified on command-line
	}
	defer func() {
		if dataReader != nil {
			err = multierr.Append(err, dataReader.Close())
		}
	}()

	transport, err := makeHTTPClient(f, isSecure, bufcurl.GetAuthority(endpointURL, requestHeaders), container.VerbosePrinter())
	if err != nil {
		return err
	}

	output := container.Stdout()
	if f.Output != "" {
		output, err = os.Create(f.Output)
		if err != nil {
			return bufcurl.ErrorHasFilename(err, f.Output)
		}
	}

	var res bufcurl.Resolver
	if f.Reflect {
		reflectHeaders, _, err := bufcurl.LoadHeaders(f.ReflectHeaders, "", requestHeaders)
		if err != nil {
			return err
		}
		reflectProtocol, err := bufcurl.ParseReflectProtocol(f.ReflectProtocol)
		if err != nil {
			return err
		}
		var closeRes func()
		res, closeRes = bufcurl.NewServerReflectionResolver(ctx, transport, clientOptions, baseURL, reflectProtocol, reflectHeaders, container.VerbosePrinter())
		defer closeRes()
	} else {
		ref, err := buffetch.NewRefParser(container.Logger(), buffetch.RefParserWithProtoFileRefAllowed()).GetRef(ctx, f.Schema)
		if err != nil {
			return err
		}
		storageosProvider := bufcli.NewStorageosProvider(false)
		// TODO: Ideally, we'd use our verbose client for this Connect client, so we can see the same
		//   kind of output in verbose mode as we see for reflection requests.
		clientConfig, err := bufcli.NewConnectClientConfig(container)
		if err != nil {
			return err
		}
		imageConfigReader, err := bufcli.NewWireImageConfigReader(
			container,
			storageosProvider,
			command.NewRunner(),
			clientConfig,
		)
		if err != nil {
			return err
		}
		imageConfigs, fileAnnotations, err := imageConfigReader.GetImageConfigs(
			ctx,
			container,
			ref,
			"",
			nil,
			nil,
			false, // input files must exist
			false, // we must include source info for generation
		)
		if err != nil {
			return err
		}
		if len(fileAnnotations) > 0 {
			if err := bufanalysis.PrintFileAnnotations(container.Stderr(), fileAnnotations, bufanalysis.FormatText.String()); err != nil {
				return err
			}
			return bufcli.ErrFileAnnotation
		}
		images := make([]bufimage.Image, 0, len(imageConfigs))
		for _, imageConfig := range imageConfigs {
			images = append(images, imageConfig.Image())
		}
		image, err := bufimage.MergeImages(images...)
		if err != nil {
			return err
		}
		res, err = bufcurl.NewImageResolver(image)
		if err != nil {
			return err
		}
	}

	methodDescriptor, err := bufcurl.ResolveMethodDescriptor(res, service, method)
	if err != nil {
		return err
	}

	// Now we can finally issue the RPC
	invoker := bufcurl.NewInvoker(container, methodDescriptor, res, transport, clientOptions, container.Arg(0), output)
	return invoker.Invoke(ctx, dataSource, dataReader, requestHeaders)
}

func makeHTTPClient(f *flags, isSecure bool, authority string, printer verbose.Printer) (connect.HTTPClient, error) {
	var dialer net.Dialer
	if f.ConnectTimeoutSeconds != 0 {
		dialer.Timeout = secondsToDuration(f.ConnectTimeoutSeconds)
	}
	if f.NoKeepAlive {
		dialer.KeepAlive = -1
	} else {
		dialer.KeepAlive = secondsToDuration(f.KeepAliveTimeSeconds)
	}
	var dialFunc func(ctx context.Context, network, address string) (net.Conn, error)
	if f.UnixSocket != "" {
		dialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			printer.Printf("* Dialing unix socket %s...", f.UnixSocket)
			return dialer.DialContext(ctx, "unix", f.UnixSocket)
		}
	} else {
		dialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
			printer.Printf("* Dialing (%s) %s...", network, address)
			conn, err := dialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			printer.Printf("* Connected to %s", conn.RemoteAddr().String())
			return conn, err
		}
	}

	var transport http.RoundTripper
	if !isSecure && f.HTTP2PriorKnowledge {
		transport = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return dialFunc(ctx, network, addr)
			},
		}
	} else {
		var tlsConfig *tls.Config
		if isSecure {
			var err error
			tlsConfig, err = bufcurl.MakeVerboseTLSConfig(&bufcurl.TLSSettings{
				KeyFile:    f.Key,
				CertFile:   f.Cert,
				CACertFile: f.CACert,
				ServerName: f.ServerName,
				Insecure:   f.Insecure,
			}, authority, printer)
			if err != nil {
				return nil, err
			}
		}
		transport = &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DialContext:       dialFunc,
			ForceAttemptHTTP2: true,
			MaxIdleConns:      1,
			TLSClientConfig:   tlsConfig,
		}
	}
	return bufcurl.NewVerboseHTTPClient(transport, printer), nil
}

func secondsToDuration(secs float64) time.Duration {
	return time.Duration(float64(time.Second) * secs)
}
