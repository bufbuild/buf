module github.com/bufbuild/buf

go 1.22.0

toolchain go1.23.1

require (
	buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go v1.34.2-20240920185553-cf97df2825f6.2
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.34.2-20240717164558-a6c49f84cc0f.2
	buf.build/gen/go/bufbuild/registry/connectrpc/go v1.16.2-20240821192916-45ba72cdd479.1
	buf.build/gen/go/bufbuild/registry/protocolbuffers/go v1.34.2-20240821192916-45ba72cdd479.2
	buf.build/go/bufplugin v0.3.0
	buf.build/go/protoyaml v0.2.0
	connectrpc.com/connect v1.17.0
	connectrpc.com/otelconnect v0.7.1
	github.com/bufbuild/protocompile v0.14.1
	github.com/bufbuild/protoplugin v0.0.0-20240911180120-7bb73e41a54a
	github.com/bufbuild/protovalidate-go v0.6.5
	github.com/docker/docker v27.2.1+incompatible
	github.com/go-chi/chi/v5 v5.1.0
	github.com/gofrs/flock v0.12.1
	github.com/gofrs/uuid/v5 v5.3.0
	github.com/google/cel-go v0.21.0
	github.com/google/go-cmp v0.6.0
	github.com/google/go-containerregistry v0.20.2
	github.com/jdx/go-netrc v1.0.0
	github.com/jhump/protoreflect/v2 v2.0.0-beta.2
	github.com/klauspost/compress v1.17.9
	github.com/klauspost/pgzip v1.2.6
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/pkg/profile v1.7.0
	github.com/quic-go/quic-go v0.47.0
	github.com/rs/cors v1.11.1
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/otel v1.30.0
	go.opentelemetry.io/otel/sdk v1.30.0
	go.opentelemetry.io/otel/trace v1.30.0
	go.uber.org/atomic v1.11.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.27.0
	golang.org/x/exp v0.0.0-20240909161429-701f63a606c0
	golang.org/x/mod v0.21.0
	golang.org/x/net v0.29.0
	golang.org/x/sync v0.8.0
	golang.org/x/term v0.24.0
	golang.org/x/tools v0.25.0
	google.golang.org/protobuf v1.34.3-0.20240816073751-94ecbc261689
	gopkg.in/yaml.v3 v3.0.1
	pluginrpc.com/pluginrpc v0.3.0
)

require (
	buf.build/gen/go/pluginrpc/pluginrpc/protocolbuffers/go v1.34.2-20240828222655-5345c0a56177.2 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Microsoft/hcsshim v0.12.6 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/containerd/cgroups/v3 v3.0.3 // indirect
	github.com/containerd/containerd v1.7.22 // indirect
	github.com/containerd/continuity v0.4.3 // indirect
	github.com/containerd/errdefs v0.2.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.15.1 // indirect
	github.com/containerd/ttrpc v1.2.5 // indirect
	github.com/containerd/typeurl/v2 v2.2.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v27.2.1+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/fgprof v0.9.5 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/pprof v0.0.0-20240910150728-a0b0bb1d4134 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/mount v0.3.4 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/onsi/ginkgo/v2 v2.20.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.55.0 // indirect
	go.opentelemetry.io/otel/metric v1.30.0 // indirect
	go.uber.org/mock v0.4.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.66.2 // indirect
)
