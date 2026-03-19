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

// Package jsonrpc2 implements a JSON-RPC 2.0 client/server over an LSP
// (Content-Length framed) byte stream.
//
// It is a minimal implementation tailored for language-server use: only
// the Content-Length framing ("VS Code codec") is supported.
package jsonrpc2

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Error codes defined by JSON-RPC 2.0 spec.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Error is a JSON-RPC 2.0 response error.
type Error struct {
	Code    int64            `json:"code"`
	Message string           `json:"message"`
	Data    *json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc2: code %d message: %s", e.Code, e.Message)
}

// ErrClosed indicates that the connection is closed.
var ErrClosed = errors.New("jsonrpc2: connection is closed")

// ID is a JSON-RPC 2.0 request ID (number or string).
type ID struct {
	Num      uint64
	Str      string
	IsString bool
}

// String returns a human-readable representation of the ID.
func (id ID) String() string {
	if id.IsString {
		return strconv.Quote(id.Str)
	}
	return strconv.FormatUint(id.Num, 10)
}

// MarshalJSON implements json.Marshaler.
func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsString {
		return json.Marshal(id.Str)
	}
	return json.Marshal(id.Num)
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *ID) UnmarshalJSON(data []byte) error {
	var n uint64
	if err := json.Unmarshal(data, &n); err == nil {
		*id = ID{Num: n}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*id = ID{Str: s, IsString: true}
	return nil
}

// Request is an incoming JSON-RPC 2.0 request or notification.
type Request struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params,omitempty"`
	ID     ID               `json:"id"`
	Notif  bool             `json:"-"` // true if this is a notification (no id)
}

// wireRequest is used for JSON marshaling (adds jsonrpc field).
type wireRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params,omitempty"`
	ID      *ID              `json:"id,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Request) UnmarshalJSON(data []byte) error {
	// Use a map to detect presence/absence of "id".
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if m, ok := raw["method"]; ok {
		if err := json.Unmarshal(m, &r.Method); err != nil {
			return err
		}
	}
	if p, ok := raw["params"]; ok {
		r.Params = &p
	}
	if idRaw, ok := raw["id"]; ok {
		if err := json.Unmarshal(idRaw, &r.ID); err != nil {
			return err
		}
		r.Notif = false
	} else {
		r.Notif = true
	}
	return nil
}

// response is an outgoing JSON-RPC 2.0 response.
type response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      ID               `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *Error           `json:"error,omitempty"`
}

// incomingResponse is the wire format for a response we receive.
type incomingResponse struct {
	ID     ID               `json:"id"`
	Result *json.RawMessage `json:"result,omitempty"`
	Error  *Error           `json:"error,omitempty"`
}

// Caller is the minimal interface for sending RPC calls and notifications.
// Both *Conn and any logging/tracing wrapper implement this.
type Caller interface {
	Call(ctx context.Context, method string, params, result any) error
	Notify(ctx context.Context, method string, params any) error
}

// compile-time check
var _ Caller = (*Conn)(nil)

// Handler handles incoming JSON-RPC requests.
type Handler interface {
	Handle(ctx context.Context, conn *Conn, req *Request)
}

// HandlerFunc adapts a function to the Handler interface. The function returns
// (result, error); the Conn automatically sends the appropriate response.
type HandlerFunc func(ctx context.Context, conn *Conn, req *Request) (any, error)

// Handle implements Handler by calling the underlying function.
func (f HandlerFunc) Handle(ctx context.Context, conn *Conn, req *Request) {
	result, err := f(ctx, conn, req)
	if req.Notif {
		return // notifications don't get responses
	}
	if err != nil {
		var rpcErr *Error
		if !errors.As(err, &rpcErr) {
			rpcErr = &Error{Code: CodeInternalError, Message: err.Error()}
		}
		_ = conn.sendResponse(&response{JSONRPC: "2.0", ID: req.ID, Error: rpcErr})
		return
	}
	raw, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		_ = conn.sendResponse(&response{
			JSONRPC: "2.0", ID: req.ID,
			Error: &Error{Code: CodeInternalError, Message: marshalErr.Error()},
		})
		return
	}
	rm := json.RawMessage(raw)
	_ = conn.sendResponse(&response{JSONRPC: "2.0", ID: req.ID, Result: &rm})
}

// GoHandler returns a Handler that handles each request in a new goroutine.
// This allows the read loop to keep processing messages while requests are
// handled concurrently.
func GoHandler(h Handler) Handler {
	return goHandler{h: h}
}

type goHandler struct {
	h Handler
}

func (g goHandler) Handle(ctx context.Context, conn *Conn, req *Request) {
	if req.Notif {
		// Notifications don't need responses; run async to not block read loop.
		go g.h.Handle(ctx, conn, req)
		return
	}
	// For requests, we must send a response. Run the handler in a goroutine;
	// the inner handler (e.g. HandlerFunc) is responsible for sending the response.
	go g.h.Handle(ctx, conn, req)
}

// CancelHandler returns a Handler that intercepts $/cancelRequest notifications
// and cancels the context of the matching in-flight request. All other requests
// are passed to the inner handler with a cancellable context.
func CancelHandler(inner Handler) Handler {
	return &cancelHandler{
		cancels: make(map[uint64]context.CancelFunc),
		inner:   inner,
	}
}

type cancelHandler struct {
	mu      sync.Mutex
	cancels map[uint64]context.CancelFunc
	inner   Handler
}

func (h *cancelHandler) Handle(ctx context.Context, conn *Conn, req *Request) {
	if req.Method == "$/cancelRequest" {
		if req.Params != nil {
			var params struct {
				ID json.RawMessage `json:"id"`
			}
			if err := json.Unmarshal(*req.Params, &params); err == nil {
				var id ID
				if err := json.Unmarshal(params.ID, &id); err == nil {
					h.mu.Lock()
					if cancel, ok := h.cancels[id.Num]; ok {
						cancel()
						delete(h.cancels, id.Num)
					}
					h.mu.Unlock()
				}
			}
		}
		return // notification, no response
	}

	if !req.Notif {
		cancelCtx, cancel := context.WithCancel(ctx)
		h.mu.Lock()
		h.cancels[req.ID.Num] = cancel
		h.mu.Unlock()
		ctx = cancelCtx
		defer func() {
			h.mu.Lock()
			delete(h.cancels, req.ID.Num)
			h.mu.Unlock()
			cancel()
		}()
	}

	h.inner.Handle(ctx, conn, req)
}

// Conn is a bidirectional JSON-RPC 2.0 connection.
type Conn struct {
	r    *bufio.Reader
	wc   io.WriteCloser
	h    Handler
	wmu  sync.Mutex // guards writes
	mu   sync.Mutex
	seq  uint64
	pend map[uint64]*pending
	done chan struct{}
	once sync.Once
	err  error // set when the connection closes
}

type pending struct {
	ch chan *incomingResponse
}

// NewConn creates a new JSON-RPC connection over the given stream. It
// immediately starts reading messages in a background goroutine. The handler
// is called for each incoming request.
func NewConn(ctx context.Context, rwc io.ReadWriteCloser, h Handler) *Conn {
	c := &Conn{
		r:    bufio.NewReaderSize(rwc, 4096),
		wc:   rwc,
		h:    h,
		pend: make(map[uint64]*pending),
		done: make(chan struct{}),
	}
	go c.readLoop(ctx)
	return c
}

// Close closes the connection.
func (c *Conn) Close() error {
	c.once.Do(func() { close(c.done) })
	return c.wc.Close()
}

// Done returns a channel that is closed when the connection is closed.
func (c *Conn) Done() <-chan struct{} {
	return c.done
}

// DisconnectNotify returns a channel that is closed when the connection is
// closed (either by Close or by the remote end).
func (c *Conn) DisconnectNotify() <-chan struct{} {
	return c.done
}

// Err returns the error that caused the connection to close, if any.
// It is only valid to call Err after Done() has been closed.
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Call sends a request and waits for the response. result should be a pointer.
func (c *Conn) Call(ctx context.Context, method string, params, result any) error {
	c.mu.Lock()
	id := c.seq
	c.seq++
	p := &pending{ch: make(chan *incomingResponse, 1)}
	c.pend[id] = p
	c.mu.Unlock()

	raw, err := json.Marshal(params)
	if err != nil {
		c.mu.Lock()
		delete(c.pend, id)
		c.mu.Unlock()
		return err
	}
	rm := json.RawMessage(raw)
	reqID := ID{Num: id}

	if err := c.writeMessage(&wireRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  &rm,
		ID:      &reqID,
	}); err != nil {
		c.mu.Lock()
		delete(c.pend, id)
		c.mu.Unlock()
		return err
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pend, id)
		c.mu.Unlock()
		return ctx.Err()
	case resp := <-p.ch:
		if resp == nil {
			return ErrClosed
		}
		if resp.Error != nil {
			return resp.Error
		}
		if result != nil && resp.Result != nil {
			return json.Unmarshal(*resp.Result, result)
		}
		return nil
	}
}

// Notify sends a notification (no response expected).
func (c *Conn) Notify(ctx context.Context, method string, params any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	return c.writeMessage(&wireRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  &rm,
		// no ID → notification
	})
}

func (c *Conn) sendResponse(resp *response) error {
	return c.writeMessage(resp)
}

func (c *Conn) writeMessage(v any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(c.wc, header); err != nil {
		return err
	}
	_, err = c.wc.Write(data)
	return err
}

func (c *Conn) readLoop(ctx context.Context) {
	var closeErr error
	defer func() {
		c.mu.Lock()
		c.err = closeErr
		c.mu.Unlock()
		c.once.Do(func() { close(c.done) })
		// Wake all pending calls.
		c.mu.Lock()
		for id, p := range c.pend {
			close(p.ch)
			delete(c.pend, id)
		}
		c.mu.Unlock()
	}()

	for {
		data, err := readFrame(c.r)
		if err != nil {
			closeErr = err
			return
		}

		// Determine if this is a request or response by checking for "method".
		var probe struct {
			Method *string          `json:"method"`
			ID     *json.RawMessage `json:"id"`
		}
		if err := json.Unmarshal(data, &probe); err != nil {
			continue
		}

		if probe.Method != nil {
			// It's a request or notification.
			var req Request
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}
			c.h.Handle(ctx, c, &req)
		} else {
			// It's a response.
			var resp incomingResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				continue
			}
			c.mu.Lock()
			p := c.pend[resp.ID.Num]
			delete(c.pend, resp.ID.Num)
			c.mu.Unlock()
			if p != nil {
				p.ch <- &resp
			}
		}
	}
}

// readFrame reads one Content-Length-framed message from r.
func readFrame(r *bufio.Reader) ([]byte, error) {
	var contentLength int
	var hasContentLength bool
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // end of headers
		}
		if after, ok := strings.CutPrefix(line, "Content-Length: "); ok {
			n, err := strconv.Atoi(after)
			if err != nil {
				return nil, fmt.Errorf("bad Content-Length: %w", err)
			}
			contentLength = n
			hasContentLength = true
		}
		// ignore other headers (Content-Type, etc.)
	}
	if !hasContentLength {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}
