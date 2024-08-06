module github.com/bufbuild/buf

go 1.22

toolchain go1.22.5

require (
	buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go v1.34.2-20240806011522-d72da8527300.2
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.34.2-20240717164558-a6c49f84cc0f.2
	buf.build/gen/go/bufbuild/registry/connectrpc/go v1.16.2-20240801134127-09fbc17f7c9e.1
	buf.build/gen/go/bufbuild/registry/protocolbuffers/go v1.34.2-20240801134127-09fbc17f7c9e.2
	connectrpc.com/connect v1.16.2
	connectrpc.com/otelconnect v0.7.1
	github.com/bufbuild/bufplugin-go v0.0.0-20240806024805-fdf7f6a32a99
	github.com/bufbuild/pluginrpc-go v0.0.0-20240806025323-5bb5ab69dd9d
	github.com/bufbuild/protocompile v0.14.0
	github.com/bufbuild/protoplugin v0.0.0-20240323223605-e2735f6c31ee
	github.com/bufbuild/protovalidate-go v0.6.3
	github.com/bufbuild/protoyaml-go v0.1.10
	github.com/docker/docker v27.1.1+incompatible
	github.com/go-chi/chi/v5 v5.1.0
	github.com/gofrs/flock v0.12.1
	github.com/gofrs/uuid/v5 v5.2.0
	github.com/google/cel-go v0.21.0
	github.com/google/go-cmp v0.6.0
	github.com/google/go-containerregistry v0.20.1
	github.com/jdx/go-netrc v1.0.0
	github.com/jhump/protoreflect v1.16.0
	github.com/klauspost/compress v1.17.9
	github.com/klauspost/pgzip v1.2.6
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/pkg/profile v1.7.0
	github.com/rs/cors v1.11.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/sdk v1.28.0
	go.opentelemetry.io/otel/trace v1.28.0
	go.uber.org/atomic v1.11.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.25.0
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56
	golang.org/x/mod v0.20.0
	golang.org/x/net v0.27.0
	golang.org/x/sync v0.8.0
	golang.org/x/term v0.22.0
	golang.org/x/tools v0.23.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	buf.build/gen/go/bufbuild/pluginrpc/protocolbuffers/go v1.34.2-20240806002831-fe8c53933c68.2 // indirect
	buf.build/gen/go/bufbuild/protovalidate/connectrpc/go v1.16.2-20240717164558-a6c49f84cc0f.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.15.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v27.1.1+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/felixge/fgprof v0.9.4 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/pprof v0.0.0-20240727154555-813a5fbdbec8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240805194559-2c9e96a0b5d4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240805194559-2c9e96a0b5d4 // indirect
)

replace github.com/bufbuild/bufplugin-go => ../bufplugin-go
