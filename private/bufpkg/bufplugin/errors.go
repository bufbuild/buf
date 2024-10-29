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

package bufplugin

import (
	"strings"

	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

// DigestMismatchError is the error returned if the Digest of a downloaded Module or Commit
// does not match the expected digest in a buf.lock file.
type DigestMismatchError struct {
	PluginFullName PluginFullName
	CommitID       uuid.UUID
	ExpectedDigest Digest
	ActualDigest   Digest
}

// Error implements the error interface.
func (m *DigestMismatchError) Error() string {
	if m == nil {
		return ""
	}
	var builder strings.Builder
	_, _ = builder.WriteString(`*** Digest verification failed`)
	if m.PluginFullName != nil {
		_, _ = builder.WriteString(` for "`)
		_, _ = builder.WriteString(m.PluginFullName.String())
		if m.CommitID != uuid.Nil {
			_, _ = builder.WriteString(`:`)
			_, _ = builder.WriteString(uuidutil.ToDashless(m.CommitID))
		}
		_, _ = builder.WriteString(`"`)
	}
	_, _ = builder.WriteString(` ***`)
	_, _ = builder.WriteString("\n")
	if m.ExpectedDigest != nil && m.ActualDigest != nil {
		_, _ = builder.WriteString("\t")
		_, _ = builder.WriteString(`Expected digest (from buf.lock): "`)
		_, _ = builder.WriteString(m.ExpectedDigest.String())
		_, _ = builder.WriteString(`"`)
		_, _ = builder.WriteString("\n")
		_, _ = builder.WriteString("\t")
		_, _ = builder.WriteString(`Actual digest: "`)
		_, _ = builder.WriteString(m.ActualDigest.String())
		_, _ = builder.WriteString(`"`)
		_, _ = builder.WriteString("\n")
	}
	_, _ = builder.WriteString("\t")
	_, _ = builder.WriteString(`This may be the result of a hand-edited or corrupted buf.lock file, a corrupted local cache, and/or an attack.`)
	_, _ = builder.WriteString("\n")
	_, _ = builder.WriteString("\t")
	_, _ = builder.WriteString(`To clear your local cache, run "buf registry cc".`)
	return builder.String()
}
