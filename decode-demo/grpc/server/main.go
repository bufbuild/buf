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

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	petv1 "github.com/bufbuild/buf/decode-demo/gen/proto/go/pet/v1"
	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	listenOn := "127.0.0.1:8080"
	listener, err := net.Listen("tcp", listenOn)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listenOn, err)
	}

	closeChan := make(chan struct{})
	codec, err := bufreflect.NewCodec()
	if err != nil {
		return fmt.Errorf("failed to construct the bufreflect.Codec: %w", err)
	}
	teeCodec := &teeCodec{
		Codec:     codec,
		closeChan: closeChan,
	}
	encoding.RegisterCodec(teeCodec)

	server := grpc.NewServer()
	petv1.RegisterPetStoreServiceServer(server, &petStoreServiceServer{})
	go func() {
		select {
		case <-closeChan:
			server.GracefulStop()
		}
	}()
	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

// teeCodec wraps the bufreflect.Codec and writes the response bytes to stdout
// for further introspection.
type teeCodec struct {
	*bufreflect.Codec

	closeChan chan struct{}
}

func (c *teeCodec) Marshal(value interface{}) ([]byte, error) {
	bytes, err := c.Codec.Marshal(value)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stdout.Write(bytes); err != nil {
		return nil, err
	}
	close(c.closeChan)
	return bytes, nil
}

// petStoreServiceServer implements the PetStoreService API.
type petStoreServiceServer struct {
	petv1.UnimplementedPetStoreServiceServer
}

// PutPet adds the pet associated with the given request into the PetStore.
func (s *petStoreServiceServer) PutPet(ctx context.Context, req *petv1.PutPetRequest) (*petv1.PutPetResponse, error) {
	return &petv1.PutPetResponse{
		Pet: &petv1.Pet{
			PetType: req.PetType,
			Name:    req.Name,
		},
	}, nil
}
