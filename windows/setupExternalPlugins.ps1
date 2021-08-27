$protocGenGoVersion = 'v1.27.1'
$protocGenGoGRPCVersion = '30dfb4b933a50fd366d7ed36ed4f71dbba2d382e'

go install google.golang.org/protobuf/cmd/protoc-gen-go@$protocGenGoVersion
go get google.golang.org/grpc@$protocGenGoGRPCVersion
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$protocGenGoGRPCVersion
