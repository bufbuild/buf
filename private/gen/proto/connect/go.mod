module buf.build/gen/go/bufbuild/buf/bufbuild/connect-go

go 1.20

replace buf.build/gen/go/bufbuild/buf/protocolbuffers/go => ../go

require (
	buf.build/gen/go/bufbuild/buf/protocolbuffers/go v1.29.0-20230303213111-ac270b5c02be.1
	github.com/bufbuild/connect-go v1.5.2
)

require google.golang.org/protobuf v1.29.1 // indirect
