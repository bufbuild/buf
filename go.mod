module github.com/bufbuild/buf

go 1.24

toolchain go1.24.4

require (
	buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go v1.36.6-20250121211742-6d880cc6cc8d.1
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.6-20250625184727-c923a0c2a132.1
	buf.build/gen/go/bufbuild/registry/connectrpc/go v1.18.1-20250616221922-7d6913ad2095.1
	buf.build/gen/go/bufbuild/registry/protocolbuffers/go v1.36.6-20250616221922-7d6913ad2095.1
	buf.build/go/app v0.1.0
	buf.build/go/bufplugin v0.9.0
	buf.build/go/protovalidate v0.13.1
	buf.build/go/protoyaml v0.6.0
	buf.build/go/spdx v0.2.0
	buf.build/go/standard v0.1.0
	connectrpc.com/connect v1.18.1
	connectrpc.com/otelconnect v0.7.2
	github.com/bufbuild/protocompile v0.14.1
	github.com/bufbuild/protoplugin v0.0.0-20250218205857-750e09ce93e1
	github.com/docker/docker v28.3.1+incompatible
	github.com/go-chi/chi/v5 v5.2.2
	github.com/gofrs/flock v0.12.1
	github.com/google/cel-go v0.25.0
	github.com/google/go-cmp v0.7.0
	github.com/google/go-containerregistry v0.20.6
	github.com/google/uuid v1.6.0
	github.com/jdx/go-netrc v1.0.0
	github.com/jhump/protoreflect/v2 v2.0.0-beta.2
	github.com/klauspost/compress v1.18.0
	github.com/klauspost/pgzip v1.2.6
	github.com/mattn/go-colorable v0.1.14
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/quic-go/quic-go v0.53.0
	github.com/rs/cors v1.11.1
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/stretchr/testify v1.10.0
	github.com/tetratelabs/wazero v1.9.0
	go.lsp.dev/jsonrpc2 v0.10.0
	go.lsp.dev/protocol v0.12.0
	go.lsp.dev/uri v0.3.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.39.0
	golang.org/x/mod v0.25.0
	golang.org/x/net v0.41.0
	golang.org/x/sync v0.15.0
	golang.org/x/term v0.32.0
	golang.org/x/tools v0.34.0
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v3 v3.0.1
	pluginrpc.com/pluginrpc v0.5.0
)

require (
	buf.build/gen/go/pluginrpc/pluginrpc/protocolbuffers/go v1.36.6-20241007202033-cf42259fcbfc.1 // indirect
	buf.build/go/interrupt v1.1.0 // indirect
	cel.dev/expr v0.24.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v28.3.1+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.3 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/segmentio/encoding v0.5.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/vbatts/tar-split v0.12.1 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.0 // indirect
	go.uber.org/mock v0.5.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.72.2 // indirect
)
