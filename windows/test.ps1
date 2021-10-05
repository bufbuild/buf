$protocVersion = '3.18.1'
$protocGenGoVersion = 'v1.27.1'
$protocGenGoGRPCVersion = '30dfb4b933a50fd366d7ed36ed4f71dbba2d382e'

choco install --confirm curl zip
curl -sSL https://github.com/protocolbuffers/protobuf/releases/download/v$protocVersion/protoc-$protocVersion-win64.zip -o protoc.zip
unzip protoc.zip
New-Item -ItemType Directory -Path C:\Users\runneradmin\protoc\bin -Force
Move-Item -Path bin\protoc.exe -Destination C:\Users\runneradmin\protoc\bin;
New-Item -ItemType Directory -Path C:\Users\runneradmin\protoc\lib\include\google\protobuf -Force
Move-Item -Path include\google\protobuf\* -Destination C:\Users\runneradmin\protoc\lib\include\google\protobuf;
$env:Path += ";C:\Users\runneradmin\protoc\bin"
Get-Command protoc.exe
go install google.golang.org/protobuf/cmd/protoc-gen-go@$protocGenGoVersion
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$protocGenGoGRPCVersion
go install .\private\buf\cmd\buf\command\protoc\internal\protoc-gen-insertion-point-writer
go install .\private\buf\cmd\buf\command\protoc\internal\protoc-gen-insertion-point-receiver
go install .\cmd\buf
go test ./...
