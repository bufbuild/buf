// Copyright 2020-2026 Buf Technologies, Inc.
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

package jsonrpc2

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// frame builds a Content-Length–framed message for use in tests.
func frame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func TestReadFrame(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "normal message",
			input: frame(`{"jsonrpc":"2.0","method":"foo"}`),
			want:  `{"jsonrpc":"2.0","method":"foo"}`,
		},
		{
			name:  "zero content-length",
			input: "Content-Length: 0\r\n\r\n",
			want:  "",
		},
		{
			name:  "extra headers ignored",
			input: "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\nContent-Length: 2\r\n\r\n{}",
			want:  "{}",
		},
		{
			name:  "reads only first message",
			input: frame(`{"id":1}`) + frame(`{"id":2}`),
			want:  `{"id":1}`,
		},
		{
			name:    "missing content-length",
			input:   "Content-Type: application/vscode-jsonrpc\r\n\r\n",
			wantErr: "missing Content-Length header",
		},
		{
			name:    "bad content-length value",
			input:   "Content-Length: abc\r\n\r\n",
			wantErr: "bad Content-Length",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := readFrame(r)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIDJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		id   ID
		json string
	}{
		{"numeric", ID{Num: 42}, "42"},
		{"zero numeric", ID{Num: 0}, "0"},
		{"string", ID{Str: "abc", IsString: true}, `"abc"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.id)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.json {
				t.Fatalf("marshal: got %s, want %s", got, tt.json)
			}
			var id ID
			if err := json.Unmarshal(got, &id); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if id != tt.id {
				t.Fatalf("roundtrip: got %+v, want %+v", id, tt.id)
			}
		})
	}
}

func TestRequestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		wantMethod string
		wantIDNum  uint64
		wantNotif  bool
	}{
		{
			name:       "request with numeric id and object params",
			input:      `{"jsonrpc":"2.0","id":1,"method":"textDocument/hover","params":{}}`,
			wantMethod: "textDocument/hover",
			wantIDNum:  1,
			wantNotif:  false,
		},
		{
			name:       "request with array params",
			input:      `{"jsonrpc":"2.0","id":2,"method":"subtract","params":[42,23]}`,
			wantMethod: "subtract",
			wantIDNum:  2,
			wantNotif:  false,
		},
		{
			name:       "notification (no id)",
			input:      `{"jsonrpc":"2.0","method":"initialized","params":{}}`,
			wantMethod: "initialized",
			wantNotif:  true,
		},
		{
			name:       "notification with no params",
			input:      `{"jsonrpc":"2.0","method":"exit"}`,
			wantMethod: "exit",
			wantNotif:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req Request
			if err := json.Unmarshal([]byte(tt.input), &req); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if req.Method != tt.wantMethod {
				t.Errorf("method: got %q, want %q", req.Method, tt.wantMethod)
			}
			if req.Notif != tt.wantNotif {
				t.Errorf("notif: got %v, want %v", req.Notif, tt.wantNotif)
			}
			if !tt.wantNotif && req.ID.Num != tt.wantIDNum {
				t.Errorf("id: got %d, want %d", req.ID.Num, tt.wantIDNum)
			}
		})
	}
}

// newTestPair creates a connected client/server Conn pair over a net.Pipe.
func newTestPair(t *testing.T, serverHandler Handler) (client, server *Conn) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	nullHandler := HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		return nil, nil
	})
	server = NewConn(t.Context(), serverConn, serverHandler)
	client = NewConn(t.Context(), clientConn, nullHandler)
	t.Cleanup(func() {
		client.Close()
		server.Close()
	})
	return client, server
}

func TestConnCall(t *testing.T) {
	t.Parallel()
	handler := HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		return map[string]string{"echo": req.Method}, nil
	})
	client, _ := newTestPair(t, handler)

	var result map[string]string
	if err := client.Call(t.Context(), "textDocument/hover", map[string]any{"x": 1}, &result); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result["echo"] != "textDocument/hover" {
		t.Errorf("got %v, want echo=textDocument/hover", result)
	}
}

func TestConnNotify(t *testing.T) {
	t.Parallel()
	received := make(chan string, 1)
	handler := HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		if req.Notif {
			received <- req.Method
		}
		return nil, nil
	})
	client, _ := newTestPair(t, handler)

	if err := client.Notify(t.Context(), "initialized", map[string]any{}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	select {
	case method := <-received:
		if method != "initialized" {
			t.Errorf("got %q, want %q", method, "initialized")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for notification")
	}
}

func TestConnErrorResponse(t *testing.T) {
	t.Parallel()
	handler := HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		return nil, &Error{Code: CodeMethodNotFound, Message: "method not found"}
	})
	client, _ := newTestPair(t, handler)

	err := client.Call(t.Context(), "unknown/method", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rpcErr *Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if rpcErr.Code != CodeMethodNotFound {
		t.Errorf("code: got %d, want %d", rpcErr.Code, CodeMethodNotFound)
	}
}

func TestConnDisconnectNotify(t *testing.T) {
	t.Parallel()
	client, server := newTestPair(t, HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		return nil, nil
	}))

	server.Close()

	select {
	case <-client.DisconnectNotify():
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for client to observe disconnect")
	}
}

func TestConnCallErrClosed(t *testing.T) {
	t.Parallel()
	testDone := make(chan struct{})
	t.Cleanup(func() { close(testDone) })

	received := make(chan struct{}, 1)
	handler := HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		select {
		case received <- struct{}{}:
		default:
		}
		<-testDone // block without responding until test cleanup
		return nil, nil
	})
	client, server := newTestPair(t, handler)

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Call(t.Context(), "foo", nil, nil)
	}()

	// Wait until the server has received the request, then drop the connection.
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive request")
	}
	server.Close()

	select {
	case err := <-errCh:
		if !errors.Is(err, ErrClosed) {
			t.Errorf("expected ErrClosed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Call to return ErrClosed")
	}
}

func TestConnInternalErrorWrapping(t *testing.T) {
	t.Parallel()
	// Per spec §5.1: if the handler returns a non-*Error, it must be wrapped
	// as a CodeInternalError response.
	handler := HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		return nil, errors.New("something went wrong")
	})
	client, _ := newTestPair(t, handler)

	err := client.Call(t.Context(), "foo", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rpcErr *Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if rpcErr.Code != CodeInternalError {
		t.Errorf("code: got %d, want %d (CodeInternalError)", rpcErr.Code, CodeInternalError)
	}
}

func TestConnErrorWithData(t *testing.T) {
	t.Parallel()
	// Per spec §5.1: error objects may include a data field.
	data := json.RawMessage(`{"detail":"extra info"}`)
	handler := HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		return nil, &Error{Code: CodeInvalidParams, Message: "invalid params", Data: &data}
	})
	client, _ := newTestPair(t, handler)

	err := client.Call(t.Context(), "foo", nil, nil)
	var rpcErr *Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if rpcErr.Code != CodeInvalidParams {
		t.Errorf("code: got %d, want %d", rpcErr.Code, CodeInvalidParams)
	}
	if rpcErr.Data == nil {
		t.Fatal("expected non-nil Data field")
	}
	if string(*rpcErr.Data) != `{"detail":"extra info"}` {
		t.Errorf("data: got %s, want %s", *rpcErr.Data, `{"detail":"extra info"}`)
	}
}

func TestConnNotifyNoResponse(t *testing.T) {
	t.Parallel()
	// Notifications (req.Notif=true) must never elicit a response per spec §4.
	// We verify by sending a notification and then a call; the call response
	// must arrive (not be displaced by a spurious notification response).
	handler := HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		return map[string]string{"method": req.Method}, nil
	})
	client, _ := newTestPair(t, handler)

	if err := client.Notify(t.Context(), "$/cancelRequest", map[string]any{"id": 1}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	var result map[string]string
	if err := client.Call(t.Context(), "textDocument/definition", nil, &result); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result["method"] != "textDocument/definition" {
		t.Errorf("got %v, want method=textDocument/definition", result)
	}
}

func TestConnConcurrentCalls(t *testing.T) {
	t.Parallel()
	// Multiple in-flight calls must be matched to their responses by ID, even
	// when responses arrive out of order. Each goroutine records the method it
	// sent and the method the server echoed back; they must match.
	handler := HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		return map[string]string{"method": req.Method}, nil
	})
	client, _ := newTestPair(t, handler)

	const n = 20
	type callResult struct {
		sent     string
		received string
		err      error
	}
	results := make(chan callResult, n)
	methods := make([]string, n)
	for i := range n {
		methods[i] = fmt.Sprintf("method/%d", i)
	}
	for _, method := range methods {
		go func() {
			var result map[string]string
			err := client.Call(t.Context(), method, nil, &result)
			results <- callResult{sent: method, received: result["method"], err: err}
		}()
	}
	for range n {
		r := <-results
		if r.err != nil {
			t.Errorf("Call %q error: %v", r.sent, r.err)
			continue
		}
		if r.sent != r.received {
			t.Errorf("ID mismatch: sent %q, got back %q", r.sent, r.received)
		}
	}
}

func TestConnCallContextCancel(t *testing.T) {
	t.Parallel()
	testDone := make(chan struct{})
	t.Cleanup(func() { close(testDone) })

	received := make(chan struct{}, 1)
	handler := HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		select {
		case received <- struct{}{}:
		default:
		}
		<-testDone
		return nil, nil
	})
	client, _ := newTestPair(t, handler)

	callCtx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Call(callCtx, "foo", nil, nil)
	}()

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive request")
	}
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Call to return after cancel")
	}
}

func TestIDString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id   ID
		want string
	}{
		{ID{Num: 0}, "0"},
		{ID{Num: 42}, "42"},
		{ID{Str: "req-1", IsString: true}, `"req-1"`},
	}
	for _, tt := range tests {
		got := tt.id.String()
		if got != tt.want {
			t.Errorf("ID%+v.String() = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestErrorError(t *testing.T) {
	t.Parallel()
	e := &Error{Code: CodeMethodNotFound, Message: "method not found"}
	want := "jsonrpc2: code -32601 message: method not found"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConnBidirectional(t *testing.T) {
	t.Parallel()
	// Both ends of a Conn can initiate calls; the connection is symmetric.
	// This mirrors LSP server-push scenarios (window/showMessage, etc.).
	clientConn, serverConn := net.Pipe()

	clientHandler := HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		return map[string]string{"from": "client"}, nil
	})
	serverHandler := HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		return map[string]string{"from": "server"}, nil
	})

	server := NewConn(t.Context(), serverConn, serverHandler)
	client := NewConn(t.Context(), clientConn, clientHandler)
	t.Cleanup(func() {
		client.Close()
		server.Close()
	})

	var clientResult map[string]string
	if err := client.Call(t.Context(), "serverMethod", nil, &clientResult); err != nil {
		t.Fatalf("client.Call: %v", err)
	}
	if clientResult["from"] != "server" {
		t.Errorf("client.Call: got %v, want from=server", clientResult)
	}

	var serverResult map[string]string
	if err := server.Call(t.Context(), "clientMethod", nil, &serverResult); err != nil {
		t.Fatalf("server.Call: %v", err)
	}
	if serverResult["from"] != "client" {
		t.Errorf("server.Call: got %v, want from=client", serverResult)
	}
}

func TestConnHandlerUnmarshalableResult(t *testing.T) {
	t.Parallel()
	// HandlerFunc must send a CodeInternalError response if the result value
	// cannot be marshaled to JSON (e.g. contains a channel).
	handler := HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		return make(chan int), nil // channels cannot be marshaled
	})
	client, _ := newTestPair(t, handler)

	err := client.Call(t.Context(), "foo", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rpcErr *Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if rpcErr.Code != CodeInternalError {
		t.Errorf("code: got %d, want %d (CodeInternalError)", rpcErr.Code, CodeInternalError)
	}
}

func TestConnCloseIdempotent(t *testing.T) {
	t.Parallel()
	clientConn, serverConn := net.Pipe()
	conn := NewConn(t.Context(), serverConn, HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		return nil, nil
	}))
	t.Cleanup(func() { clientConn.Close() })

	// Calling Close multiple times must not panic or block.
	conn.Close()
	conn.Close()
	conn.Close()
}

func TestConnLargeMessage(t *testing.T) {
	t.Parallel()
	// Messages larger than the 4096-byte bufio buffer must be read correctly.
	handler := HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		var params map[string]string
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return params, nil
	})
	client, _ := newTestPair(t, handler)

	// Build a params object whose JSON representation exceeds 4KB.
	params := map[string]string{"data": strings.Repeat("x", 16*1024)}
	var result map[string]string
	if err := client.Call(t.Context(), "echo", params, &result); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result["data"] != params["data"] {
		t.Errorf("large message roundtrip failed: got %d bytes, want %d", len(result["data"]), len(params["data"]))
	}
}

func TestConnCallUnmarshalableParams(t *testing.T) {
	t.Parallel()
	// Call must clean up the pending entry and return an error if params
	// cannot be marshaled, without leaving a leaked entry in c.pend.
	client, _ := newTestPair(t, HandlerFunc(func(_ context.Context, _ *Conn, _ *Request) (any, error) {
		return nil, nil
	}))

	// Channels cannot be marshaled to JSON.
	err := client.Call(t.Context(), "foo", make(chan int), nil)
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}

	// The connection must still be usable after the failed call.
	if err := client.Call(t.Context(), "foo", nil, nil); err != nil {
		t.Fatalf("subsequent Call failed: %v", err)
	}
}

func TestConnHandlerServerPush(t *testing.T) {
	t.Parallel()
	// A handler may call conn.Notify to push messages to the client while
	// processing a request — the core LSP server-push pattern
	// (window/showMessage, $/progress, etc.).
	pushReceived := make(chan string, 1)

	clientConn, serverConn := net.Pipe()

	clientHandler := HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		if req.Notif {
			pushReceived <- req.Method
		}
		return nil, nil
	})
	serverHandler := HandlerFunc(func(ctx context.Context, conn *Conn, req *Request) (any, error) {
		// Push a notification back to the client before responding.
		if err := conn.Notify(ctx, "window/logMessage", map[string]string{"message": "hello"}); err != nil {
			return nil, err
		}
		return "ok", nil
	})

	server := NewConn(t.Context(), serverConn, serverHandler)
	client := NewConn(t.Context(), clientConn, clientHandler)
	t.Cleanup(func() {
		client.Close()
		server.Close()
	})

	var result string
	if err := client.Call(t.Context(), "textDocument/hover", nil, &result); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result != "ok" {
		t.Errorf("result: got %q, want %q", result, "ok")
	}

	select {
	case method := <-pushReceived:
		if method != "window/logMessage" {
			t.Errorf("push method: got %q, want window/logMessage", method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server push notification")
	}
}

func TestConnMalformedJSONDropped(t *testing.T) {
	t.Parallel()
	// A frame with valid Content-Length but invalid JSON body must be silently
	// dropped; the connection must remain usable for subsequent valid messages.
	clientConn, serverConn := net.Pipe()

	received := make(chan string, 1)
	server := NewConn(t.Context(), serverConn, HandlerFunc(func(_ context.Context, _ *Conn, req *Request) (any, error) {
		received <- req.Method
		return nil, nil
	}))
	t.Cleanup(func() {
		clientConn.Close()
		server.Close()
	})

	// Write a frame whose body is not valid JSON, then a valid notification.
	badBody := "{{{{{" // 5 bytes of invalid JSON
	validMsg := `{"jsonrpc":"2.0","method":"$/ping"}`
	payload := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(badBody), badBody) +
		fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(validMsg), validMsg)
	if _, err := clientConn.Write([]byte(payload)); err != nil {
		t.Fatalf("write: %v", err)
	}

	select {
	case method := <-received:
		if method != "$/ping" {
			t.Errorf("got %q, want $/ping", method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: connection not usable after malformed JSON frame")
	}
}
