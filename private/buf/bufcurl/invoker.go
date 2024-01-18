// Copyright 2020-2024 Buf Technologies, Inc.
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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type deferredMessage struct {
	data []byte
}

type protoCodec struct{}

func (p protoCodec) Name() string {
	return "proto"
}

func (p protoCodec) Marshal(a any) ([]byte, error) {
	protoMessage, ok := a.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("cannot marshal: %T does not implement proto.Message", a)
	}
	return protoencoding.NewWireMarshaler().Marshal(protoMessage)
}

func (p protoCodec) Unmarshal(bytes []byte, a any) error {
	if deferred, ok := a.(*deferredMessage); ok {
		// must make a copy since Connect framework will re-use the byte slice
		deferred.data = make([]byte, len(bytes))
		copy(deferred.data, bytes)
		return nil
	}
	protoMessage, ok := a.(proto.Message)
	if !ok {
		return fmt.Errorf("cannot unmarshal: %T does not implement proto.Message", a)
	}
	return protoencoding.NewWireUnmarshaler(nil).Unmarshal(bytes, protoMessage)
}

type invokeClient = connect.Client[dynamicpb.Message, deferredMessage]

type invoker struct {
	md           protoreflect.MethodDescriptor
	res          protoencoding.Resolver
	emitDefaults bool
	client       *invokeClient
	output       io.Writer
	errOutput    io.Writer
	printer      verbose.Printer
}

// NewInvoker creates a new invoker for invoking the method described by the
// given descriptor. The given writer is used to write the output response(s)
// in JSON format. The given resolver is used to resolve Any messages and
// extensions that appear in the input or output. Other parameters are used
// to create a Connect client, for issuing the RPC.
func NewInvoker(container appext.Container, md protoreflect.MethodDescriptor, res protoencoding.Resolver, emitDefaults bool, httpClient connect.HTTPClient, opts []connect.ClientOption, url string, out io.Writer) Invoker {
	opts = append(opts, connect.WithCodec(protoCodec{}))
	// TODO: could also provide custom compressor implementations that could give us
	//  optics into when request and response messages are compressed (which could be
	//  useful to include in verbose output).
	return &invoker{
		md:           md,
		res:          res,
		emitDefaults: emitDefaults,
		output:       out,
		printer:      container.VerbosePrinter(),
		errOutput:    container.Stderr(),
		client:       connect.NewClient[dynamicpb.Message, deferredMessage](httpClient, url, opts...),
	}
}

func (inv *invoker) Invoke(ctx context.Context, dataSource string, data io.Reader, headers http.Header) error {
	inv.printer.Printf("* Invoking RPC %s\n", inv.md.FullName())
	// request's user-agent header(s) get overwritten by protocol, so we stash them in the
	// context so that underlying transport can restore them
	ctx = withUserAgent(ctx, headers)
	switch {
	case inv.md.IsStreamingServer() && inv.md.IsStreamingClient():
		return inv.handleBidiStream(ctx, dataSource, data, headers)
	case inv.md.IsStreamingServer():
		return inv.handleServerStream(ctx, dataSource, data, headers)
	case inv.md.IsStreamingClient():
		return inv.handleClientStream(ctx, dataSource, data, headers)
	default:
		return inv.handleUnary(ctx, dataSource, data, headers)
	}
}

func (inv *invoker) handleUnary(ctx context.Context, dataSource string, data io.Reader, headers http.Header) error {
	provider := newMessageProvider(dataSource, data, inv.res)
	msg := dynamicpb.NewMessage(inv.md.Input())
	if err := provider.next(msg); err != nil {
		return err
	}
	// make sure input does not contain a second message
	dummy := dynamicpb.NewMessage(inv.md.Input())
	if err := provider.next(dummy); err != io.EOF {
		return fmt.Errorf("method %s is a unary RPC, but input contained more than one request message", inv.md.Name())
	}

	req := connect.NewRequest(msg)
	for k, v := range headers {
		req.Header()[k] = v
	}
	resp, err := inv.client.CallUnary(ctx, req)
	if err != nil {
		var connErr *connect.Error
		if !errors.As(err, &connErr) {
			return err
		}
		err := inv.handleErrorResponse(connErr)
		return err
	}
	return inv.handleResponse(resp.Msg.data, nil)
}

func (inv *invoker) handleClientStream(ctx context.Context, dataSource string, data io.Reader, headers http.Header) (retErr error) {
	provider := newStreamMessageProvider(dataSource, data, inv.res)
	msg := dynamicpb.NewMessage(inv.md.Input())
	stream := inv.client.CallClientStream(ctx)
	for k, v := range headers {
		stream.RequestHeader()[k] = v
	}
	defer func() {
		if retErr != nil {
			var connErr *connect.Error
			if errors.As(retErr, &connErr) {
				retErr = inv.handleErrorResponse(connErr)
			}
		}
	}()
	if err, isStreamError := inv.handleStreamRequest(provider, msg, stream); err != nil {
		if isStreamError {
			_, recvErr := stream.CloseAndReceive()
			// stream.Send should return io.EOF on error, and caller is expected to call
			// stream.Receive to get the actual RPC error.
			if recvErr != nil {
				return recvErr
			}
		}
		return err
	}
	resp, err := stream.CloseAndReceive()
	if err != nil {
		return err
	}
	return inv.handleResponse(resp.Msg.data, nil)
}

func (inv *invoker) handleServerStream(ctx context.Context, dataSource string, data io.Reader, headers http.Header) (retErr error) {
	provider := newMessageProvider(dataSource, data, inv.res)
	msg := dynamicpb.NewMessage(inv.md.Input())
	if err := provider.next(msg); err != nil {
		return err
	}
	// make sure input does not contain a second message
	dummy := dynamicpb.NewMessage(inv.md.Input())
	if err := provider.next(dummy); err != io.EOF {
		return fmt.Errorf("method %s is a unary RPC, but input contained more than one request message", inv.md.Name())
	}

	req := connect.NewRequest(msg)
	for k, v := range headers {
		req.Header()[k] = v
	}
	defer func() {
		if retErr != nil {
			var connErr *connect.Error
			if errors.As(retErr, &connErr) {
				retErr = inv.handleErrorResponse(connErr)
			}
		}
	}()

	stream, err := inv.client.CallServerStream(ctx, req)
	if err != nil {
		return err
	}
	return inv.handleStreamResponse(&serverStreamAdapter{stream: stream})
}

func (inv *invoker) handleBidiStream(ctx context.Context, dataSource string, data io.Reader, headers http.Header) (retErr error) {
	ctx, cancel := context.WithCancel(ctx)
	provider := newStreamMessageProvider(dataSource, data, inv.res)
	msg := dynamicpb.NewMessage(inv.md.Input())
	stream := inv.client.CallBidiStream(ctx)
	for k, v := range headers {
		stream.RequestHeader()[k] = v
	}

	defer func() {
		if retErr != nil {
			var connErr *connect.Error
			if errors.As(retErr, &connErr) {
				retErr = inv.handleErrorResponse(connErr)
			}
		}
	}()

	var recvErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		if err := inv.handleStreamResponse(stream); err != nil {
			recvErr = err
		}
	}()
	defer func() {
		wg.Wait()
		if recvErr != nil {
			// may just get io.EOF or cancel error when trying to write to closed
			// request stream whereas actual error details will be seen on the read side
			if retErr == nil || errors.Is(retErr, io.EOF) || isCancelled(retErr) {
				retErr = recvErr
			}
		}
	}()
	shouldCancel := true
	defer func() {
		if shouldCancel {
			cancel()
		}
	}()

	err, isStreamError := inv.handleStreamRequest(provider, msg, stream)
	shouldCancel = err != nil && !isStreamError
	if err != nil {
		return err
	}
	return stream.CloseRequest()
}

func isCancelled(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	var connErr *connect.Error
	if errors.As(err, &connErr) {
		return connErr.Code() == connect.CodeCanceled
	}
	return false
}

func (inv *invoker) handleResponse(data []byte, msg *dynamicpb.Message) error {
	if msg == nil {
		msg = dynamicpb.NewMessage(inv.md.Output())
	}
	if err := protoencoding.NewWireUnmarshaler(inv.res).Unmarshal(data, msg); err != nil {
		return err
	}
	jsonMarshalerOptions := []protoencoding.JSONMarshalerOption{
		protoencoding.JSONMarshalerWithIndent(),
	}
	if inv.emitDefaults {
		jsonMarshalerOptions = append(
			jsonMarshalerOptions,
			protoencoding.JSONMarshalerWithEmitUnpopulated(),
		)
	}
	unrecognized := countUnrecognized(msg.ProtoReflect())
	if unrecognized > 0 {
		inv.printer.Printf("Response message (%s) contained %d bytes of unrecognized fields.",
			msg.ProtoReflect().Descriptor().FullName(), unrecognized)
	}
	outputBytes, err := protoencoding.NewJSONMarshaler(inv.res, jsonMarshalerOptions...).Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(inv.output, "%s\n", outputBytes)
	return err
}

type clientStream interface {
	Send(message *dynamicpb.Message) error
}

type serverStream interface {
	Receive() (*deferredMessage, error)
	CloseResponse() error
}

type serverStreamAdapter struct {
	stream *connect.ServerStreamForClient[deferredMessage]
}

func (ssa *serverStreamAdapter) Receive() (*deferredMessage, error) {
	if !ssa.stream.Receive() {
		err := ssa.stream.Err()
		if err == nil {
			err = io.EOF
		}
		return nil, err
	}
	return ssa.stream.Msg(), nil
}

func (ssa *serverStreamAdapter) CloseResponse() error {
	return ssa.stream.Close()
}

func (inv *invoker) handleStreamRequest(provider messageProvider, msg *dynamicpb.Message, stream clientStream) (error, bool) {
	for {
		if err := provider.next(msg); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err, false
		}
		if err := stream.Send(msg); err != nil {
			return err, true
		}
	}
	return nil, false
}

func (inv *invoker) handleStreamResponse(stream serverStream) (retError error) {
	defer func() {
		err := stream.CloseResponse()
		if err != nil && retError == nil {
			retError = err
		}
	}()
	msg := dynamicpb.NewMessage(inv.md.Output())
	for {
		responseMsg, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		if err := inv.handleResponse(responseMsg.data, msg); err != nil {
			return err
		}
	}
}

func (inv *invoker) handleErrorResponse(connErr *connect.Error) error {
	// NB: This is a nasty hack: we create a fake request that looks
	//     like a unary Connect request, so that the ErrorWriter will
	//     print the error in the format we want, which is just the
	//     JSON representation of the Connect error. (We don't need
	//     an end-stream message representation or for the content
	//     to be put into response headers, which is what it may
	//     choose to do if it detects a different protocol in the
	//     request).
	req := &http.Request{
		Header: http.Header{},
	}
	req.Header.Add("content-type", "application/json")

	w := connect.NewErrorWriter()
	responseWriter := httptest.NewRecorder()
	err := w.Write(responseWriter, req, connErr)
	if err != nil {
		return err
	}
	var prettyPrinted bytes.Buffer
	if err := json.Indent(&prettyPrinted, responseWriter.Body.Bytes(), "", "   "); err != nil {
		return err
	}
	_, _ = inv.errOutput.Write(prettyPrinted.Bytes())
	_, _ = inv.errOutput.Write([]byte("\n"))
	return app.NewError(int(connErr.Code()*8), "")
}

func newStreamMessageProvider(dataSource string, data io.Reader, res protoencoding.Resolver) messageProvider {
	if data == nil {
		// if no data provided, treat as empty input
		data = bytes.NewBuffer(nil)
	}
	return &streamMessageProvider{name: dataSource, dec: json.NewDecoder(data), res: res}
}

func newMessageProvider(dataSource string, data io.Reader, res protoencoding.Resolver) messageProvider {
	if data == nil {
		// if no data provider, treat as if single empty message
		return &singleEmptyMessageProvider{}
	} else {
		return newStreamMessageProvider(dataSource, data, res)
	}
}

type messageProvider interface {
	next(proto.Message) error
}

type singleEmptyMessageProvider struct {
	read bool
}

func (s *singleEmptyMessageProvider) next(_ proto.Message) error {
	if !s.read {
		s.read = true
		return nil
	}
	return io.EOF
}

type streamMessageProvider struct {
	name string
	dec  *json.Decoder
	res  protoencoding.Resolver
}

func (s *streamMessageProvider) next(msg proto.Message) error {
	var jsonData json.RawMessage
	if err := s.dec.Decode(&jsonData); err != nil {
		if err == io.EOF {
			return err
		}
		return fmt.Errorf("%s at offset %d: %w", s.name, s.dec.InputOffset(), err)
	}
	return protoencoding.NewJSONUnmarshaler(
		s.res, protoencoding.JSONUnmarshalerWithDisallowUnknown(),
	).Unmarshal(jsonData, msg)
}

func countUnrecognized(msg protoreflect.Message) int {
	var count int
	msg.Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		switch {
		case field.IsMap():
			if !isMessageKind(field.MapValue().Kind()) {
				break
			}
			// Note: Technically, each message entry could have had unrecognized field
			// bytes, but they are discarded by the runtime. So we can only look at
			// unrecognized fields in message values inside the map.
			mapVal := val.Map()
			mapVal.Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
				count += countUnrecognized(v.Message())
				return true
			})
		case field.IsList():
			if !isMessageKind(field.Kind()) {
				break
			}
			listVal := val.List()
			for i, length := 0, listVal.Len(); i < length; i++ {
				count += countUnrecognized(listVal.Get(i).Message())
			}
		case isMessageKind(field.Kind()):
			count += countUnrecognized(val.Message())
		}
		return true
	})
	return count + len(msg.GetUnknown())
}

func isMessageKind(k protoreflect.Kind) bool {
	return k == protoreflect.MessageKind || k == protoreflect.GroupKind
}
