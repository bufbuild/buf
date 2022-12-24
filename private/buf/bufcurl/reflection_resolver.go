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

package bufcurl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	reflectionv1 "buf.build/gen/go/grpc/grpc/protocolbuffers/go/grpc/reflection/v1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ReflectProtocol string

const (
	ReflectProtocolGRPCv1      = ReflectProtocol("grpc-v1")
	ReflectProtocolGRPCv1alpha = ReflectProtocol("grpc-v1alpha")
	ReflectProtocolAuto        = ReflectProtocol("auto")
)

// NewServerReflectionResolver creates a new resolver using the given details to
// create an RPC reflection client, to ask the server for descriptors.
func NewServerReflectionResolver(
	ctx context.Context,
	httpClient connect.HTTPClient,
	opts []connect.ClientOption,
	baseURL string,
	reflectVersion ReflectProtocol,
	headers http.Header,
	printer verbose.Printer,
) (r Resolver, closeResolver func()) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	var v1Client, v1alphaClient *reflectClient
	if reflectVersion != ReflectProtocolGRPCv1 {
		v1alphaClient = connect.NewClient[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse](httpClient, baseURL+"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo", opts...)
	}
	if reflectVersion != ReflectProtocolGRPCv1alpha {
		v1Client = connect.NewClient[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse](httpClient, baseURL+"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo", opts...)
	}
	// if version is neither "v1" nor "v1alpha", then we have both clients and
	// will automatically decide which one to use by trying v1 first and falling
	// back to v1alpha on "not implemented" error

	// elide the "upload finished" trace message for reflection calls
	ctx = skippingUploadFinishedMessage(ctx)
	// request's user-agent header(s) get overwritten by protocol, so we stash them in the
	// context so that underlying transport can restore them
	ctx = withUserAgent(ctx, headers)

	res := &reflectionResolver{
		ctx:              ctx,
		v1Client:         v1Client,
		v1alphaClient:    v1alphaClient,
		useV1Alpha:       reflectVersion == ReflectProtocolGRPCv1alpha,
		headers:          headers,
		printer:          printer,
		downloadedProtos: map[string]*descriptorpb.FileDescriptorProto{},
	}
	return res, res.Reset
}

type reflectClient = connect.Client[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse]
type reflectStream = connect.BidiStreamForClient[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse]

type reflectionResolver struct {
	ctx                     context.Context
	headers                 http.Header
	printer                 verbose.Printer
	v1Client, v1alphaClient *reflectClient

	mu                      sync.Mutex
	useV1Alpha              bool
	v1Stream, v1alphaStream *reflectStream
	downloadedProtos        map[string]*descriptorpb.FileDescriptorProto
	cachedFiles             protoregistry.Files
	cachedExts              protoregistry.Types
}

func (r *reflectionResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, err := r.cachedFiles.FindDescriptorByName(name)
	if d != nil {
		return d, nil
	}
	if err != protoregistry.NotFound {
		return nil, err
	}
	// if not found in existing files, fetch more
	fileDescriptorProtos, err := r.fileContainingSymbolLocked(name)
	if err != nil {
		// intentionally not using "%w" because, depending on the code, the bufcli
		// app framework might incorrectly interpret it and report a bad error message.
		return nil, fmt.Errorf("failed to resolve symbol %q: %v", name, err)
	}
	if err := r.cacheFilesLocked(fileDescriptorProtos); err != nil {
		return nil, err
	}
	// now it should definitely be in there!
	return r.cachedFiles.FindDescriptorByName(name)
}

func (r *reflectionResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	d, err := r.FindDescriptorByName(message)
	if err != nil {
		return nil, err
	}
	md, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("element %s is a %s, not a message", message, DescriptorKind(d))
	}
	return dynamicpb.NewMessageType(md), nil
}

func (r *reflectionResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	pos := strings.LastIndexByte(url, '/')
	typeName := url[pos+1:]
	return r.FindMessageByName(protoreflect.FullName(typeName))
}

func (r *reflectionResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	d, err := r.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	fd, ok := d.(protoreflect.FieldDescriptor)
	if !ok || !fd.IsExtension() {
		return nil, fmt.Errorf("element %s is a %s, not an extension", field, DescriptorKind(d))
	}
	return dynamicpb.NewExtensionType(fd), nil
}

func (r *reflectionResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ext, err := r.cachedExts.FindExtensionByNumber(message, field)
	if ext != nil {
		return ext, nil
	}
	if err != protoregistry.NotFound {
		return nil, err
	}
	// if not found in existing files, fetch more
	fileDescriptorProtos, err := r.fileContainingExtensionLocked(message, field)
	if err != nil {
		// intentionally not using "%w" because, depending on the code, the bufcli
		// app framework might incorrectly interpret it and report a bad error message.
		return nil, fmt.Errorf("failed to resolve extension %d for %q: %v", field, message, err)
	}
	if err := r.cacheFilesLocked(fileDescriptorProtos); err != nil {
		return nil, err
	}
	// now it should definitely be in there!
	return r.cachedExts.FindExtensionByNumber(message, field)
}

func (r *reflectionResolver) fileContainingSymbolLocked(name protoreflect.FullName) ([]*descriptorpb.FileDescriptorProto, error) {
	r.printer.Printf("* Using server reflection to resolve %q\n", name)
	resp, err := r.sendLocked(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: string(name),
		},
	})
	if err != nil {
		return nil, err
	}
	return descriptorsInResponse(resp)
}

func (r *reflectionResolver) fileContainingExtensionLocked(message protoreflect.FullName, field protoreflect.FieldNumber) ([]*descriptorpb.FileDescriptorProto, error) {
	r.printer.Printf("* Using server reflection to retrieve extension %d for %q\n", field, message)
	resp, err := r.sendLocked(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_FileContainingExtension{
			FileContainingExtension: &reflectionv1.ExtensionRequest{
				ContainingType:  string(message),
				ExtensionNumber: int32(field),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return descriptorsInResponse(resp)
}

func (r *reflectionResolver) fileByNameLocked(name string) ([]*descriptorpb.FileDescriptorProto, error) {
	r.printer.Printf("* Using server reflection to download file %q\n", name)
	resp, err := r.sendLocked(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_FileByFilename{
			FileByFilename: name,
		},
	})
	if err != nil {
		return nil, err
	}
	return descriptorsInResponse(resp)
}

func descriptorsInResponse(resp *reflectionv1.ServerReflectionResponse) ([]*descriptorpb.FileDescriptorProto, error) {
	switch response := resp.MessageResponse.(type) {
	case *reflectionv1.ServerReflectionResponse_ErrorResponse:
		// TODO: ideally, this would be a wire error
		//   see https://github.com/bufbuild/connect-go/issues/420
		return nil, connect.NewError(connect.Code(response.ErrorResponse.ErrorCode), errors.New(response.ErrorResponse.ErrorMessage))
	case *reflectionv1.ServerReflectionResponse_FileDescriptorResponse:
		files := make([]*descriptorpb.FileDescriptorProto, len(response.FileDescriptorResponse.FileDescriptorProto))
		for i, data := range response.FileDescriptorResponse.FileDescriptorProto {
			var file descriptorpb.FileDescriptorProto
			if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, &file); err != nil {
				return nil, err
			}
			files[i] = &file
		}
		return files, nil
	default:
		return nil, fmt.Errorf("server replied with unsupported response type: %T", resp.MessageResponse)
	}
}

func (r *reflectionResolver) cacheFilesLocked(files []*descriptorpb.FileDescriptorProto) error {
	for _, file := range files {
		if _, ok := r.downloadedProtos[file.GetName()]; ok {
			continue // already downloaded, don't bother overwriting
		}
		r.downloadedProtos[file.GetName()] = file
	}
	for _, file := range files {
		if err := r.cacheFileLocked(file.GetName(), nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *reflectionResolver) cacheFileLocked(name string, seen []string) error {
	if _, err := r.cachedFiles.FindFileByPath(name); err == nil {
		return nil // already processed this file
	}
	for i, alreadySeen := range seen {
		if name == alreadySeen {
			// we've seen this file already which means malformed
			// file descriptor protos that have an import cycle
			cycle := append(seen[i:], name)
			return fmt.Errorf("downloaded files contain an import cycle: %s", strings.Join(cycle, " -> "))
		}
	}

	file := r.downloadedProtos[name]
	if file == nil {
		// download missing file(s)
		moreFiles, err := r.fileByNameLocked(name)
		if err != nil {
			return err
		}
		for _, newFile := range moreFiles {
			r.downloadedProtos[newFile.GetName()] = newFile
			if newFile.GetName() == name {
				file = newFile
			}
		}
		if file == nil {
			return fmt.Errorf("requested file %q but response did not contain it", name)
		}
	}

	// make sure imports have been downloaded and cached
	for _, dep := range file.Dependency {
		if err := r.cacheFileLocked(dep, append(seen, name)); err != nil {
			return err
		}
	}

	// now we can create and cache this file
	fileDescriptor, err := protodesc.NewFile(file, &r.cachedFiles)
	if err != nil {
		return err
	}
	if err := r.cachedFiles.RegisterFile(fileDescriptor); err != nil {
		return err
	}
	registerExtensions(&r.cachedExts, fileDescriptor)
	r.printer.Printf("* Server reflection has resolved file %q\n", fileDescriptor.Path())
	return nil
}

func (r *reflectionResolver) sendLocked(req *reflectionv1.ServerReflectionRequest) (*reflectionv1.ServerReflectionResponse, error) {
	stream, isNew := r.getStreamLocked()
	resp, err := send(stream, req)
	if isNotImplemented(err) && !r.useV1Alpha && r.v1alphaClient != nil {
		r.resetLocked()
		r.useV1Alpha = true
		stream, isNew = r.getStreamLocked()
		resp, err = send(stream, req)
	}
	if err != nil && !isNew {
		// the existing stream broke; try again with a new stream
		r.resetLocked()
		stream, _ = r.getStreamLocked()
		resp, err = send(stream, req)
	}
	return resp, err
}

func isNotImplemented(err error) bool {
	var connErr *connect.Error
	ok := errors.As(err, &connErr)
	return ok && connErr.Code() == connect.CodeUnimplemented
}

func send(stream *reflectStream, req *reflectionv1.ServerReflectionRequest) (*reflectionv1.ServerReflectionResponse, error) {
	sendErr := stream.Send(req)
	// even if sendErr != nil, we still call Receive because Send will typically return
	// io.EOF and caller is expected to use Receive to get the RPC error result.
	resp, recvErr := stream.Receive()
	if sendErr != nil && recvErr == nil {
		return nil, sendErr
	}
	return resp, recvErr
}

func (r *reflectionResolver) getStreamLocked() (*reflectStream, bool) {
	if r.useV1Alpha {
		isNew := r.maybeCreateStreamLocked(r.v1alphaClient, &r.v1alphaStream)
		return r.v1alphaStream, isNew
	}
	isNew := r.maybeCreateStreamLocked(r.v1Client, &r.v1Stream)
	return r.v1Stream, isNew
}

func (r *reflectionResolver) maybeCreateStreamLocked(client *reflectClient, stream **reflectStream) bool {
	if *stream != nil {
		return false // already created
	}
	*stream = client.CallBidiStream(r.ctx)
	for k, v := range r.headers {
		(*stream).RequestHeader()[k] = v
	}
	return true
}

func (r *reflectionResolver) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetLocked()
}

func (r *reflectionResolver) resetLocked() {
	if r.v1Stream != nil {
		reset(r.v1Stream)
		r.v1Stream = nil
	}
	if r.v1alphaStream != nil {
		reset(r.v1alphaStream)
		r.v1alphaStream = nil
	}
}

func reset(stream *reflectStream) {
	_ = stream.CloseRequest()
	// Try to terminate gracefully by receiving the end of stream
	// (this call should return io.EOF). If we skip this and
	// immediately call CloseResponse, it could result in the
	// RPC being cancelled, which results in some nuisance
	// "cancel" errors.
	_, _ = stream.Receive()
	_ = stream.CloseResponse()
}
