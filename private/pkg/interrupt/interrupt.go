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

package interrupt

import (
	"context"
	"os/signal"
)

// Handle returns a copy of the parent [context.Context] that is marked done
// when an interrupt signal arrives or when the parent Context's Done channel
// is closed, whichever happens first.
//
// Signal handling is unregistered automatically by this function when the
// first interrupt signal arrives, which will restore the default interrupt
// signal behavior of Go programs (to exit).
//
// In effect, this function is functionally equivalent to:
//
//	ctx, cancel := signal.NotifyContext(ctx, interrupt.Signals...)
//	go func() {
//	  <-ctx.Done()
//	  cancel()
//	}()
//
// Most programs should wrap their contexts using this function to enable interrupt
// signal handling. The first interrupt signal will result in the context's Done
// channel closing. The second interrupt signal will result in the program exiting.
//
//	func main() {
//	  ctx := interrupt.Handle(context.Background())
//	  ...
//	}
func Handle(ctx context.Context) context.Context {
	ctx, cancel := signal.NotifyContext(ctx, Signals...)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return ctx
}
