$protocVersion = '3.18.0'
$protocGenGoVersion = 'v1.27.1'
$protocGenGoGRPCVersion = '30dfb4b933a50fd366d7ed36ed4f71dbba2d382e'

choco install --confirm curl zip
curl -sSL https://github.com/protocolbuffers/protobuf/releases/download/v$protocVersion/protoc-$protocVersion-win64.zip -o protoc.zip
unzip protoc.zip
# TODO: move bin/protoc to somewhere on the PATH
go install google.golang.org/protobuf/cmd/protoc-gen-go@$protocGenGoVersion
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$protocGenGoGRPCVersion
go install .\private\buf\cmd\buf\command\protoc\internal\protoc-gen-insertion-point-writer
go install .\private\buf\cmd\buf\command\protoc\internal\protoc-gen-insertion-point-receiver
go install .\cmd\buf
