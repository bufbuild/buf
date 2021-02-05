// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufprint

import (
	"context"
	"io"

	registryv1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type textUserPrinter struct {
	writer io.Writer
}

func newTextUserPrinter(writer io.Writer) *textUserPrinter {
	return &textUserPrinter{
		writer: writer,
	}
}

func (p *textUserPrinter) PrintUsers(ctx context.Context, messages ...*registryv1alpha1.User) error {
	if len(messages) == 0 {
		return nil
	}
	return WithTabWriter(
		p.writer,
		[]string{
			"ID",
			"Username",
		},
		func(tabWriter TabWriter) error {
			for _, user := range messages {
				if err := tabWriter.Write(
					user.Id,
					user.Username,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
