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

// Code generated from the LSP metaModel. DO NOT EDIT.

package lspprotocol

import (
	"encoding/json"
	"fmt"
)

// DocumentChange is a union of various file edit operations.
//
// Exactly one field of this struct is non-nil; see [DocumentChange.Valid].
//
// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#resourceChanges
type DocumentChange struct {
	TextDocumentEdit *TextDocumentEdit
	CreateFile       *CreateFile
	RenameFile       *RenameFile
	DeleteFile       *DeleteFile
}

// Valid reports whether the DocumentChange sum-type value is valid,
// that is, exactly one of create, delete, edit, or rename.
func (d DocumentChange) Valid() bool {
	n := 0
	if d.TextDocumentEdit != nil {
		n++
	}
	if d.CreateFile != nil {
		n++
	}
	if d.RenameFile != nil {
		n++
	}
	if d.DeleteFile != nil {
		n++
	}
	return n == 1
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *DocumentChange) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	if _, ok := m["textDocument"]; ok {
		d.TextDocumentEdit = new(TextDocumentEdit)
		return json.Unmarshal(data, d.TextDocumentEdit)
	}

	// The {Create,Rename,Delete}File types all share a 'kind' field.
	kind := m["kind"]
	switch kind {
	case "create":
		d.CreateFile = new(CreateFile)
		return json.Unmarshal(data, d.CreateFile)
	case "rename":
		d.RenameFile = new(RenameFile)
		return json.Unmarshal(data, d.RenameFile)
	case "delete":
		d.DeleteFile = new(DeleteFile)
		return json.Unmarshal(data, d.DeleteFile)
	}
	return fmt.Errorf("DocumentChanges: unexpected kind: %q", kind)
}

// MarshalJSON implements json.Marshaler.
func (d *DocumentChange) MarshalJSON() ([]byte, error) {
	if d.TextDocumentEdit != nil {
		return json.Marshal(d.TextDocumentEdit)
	} else if d.CreateFile != nil {
		return json.Marshal(d.CreateFile)
	} else if d.RenameFile != nil {
		return json.Marshal(d.RenameFile)
	} else if d.DeleteFile != nil {
		return json.Marshal(d.DeleteFile)
	}
	return nil, fmt.Errorf("empty DocumentChanges union value")
}
