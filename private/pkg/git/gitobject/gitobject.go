// Copyright 2020-2023 Buf Technologies, Inc.
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

package gitobject

import (
	"time"

	"github.com/bufbuild/buf/private/pkg/command"
)

// ID represents the ID of a Git object (tree, blob, or commit).
type ID interface {
	// Hex is the hexadecimal representation of this ID.
	Hex() string
	// String returns the hexadecimal representation of this ID.
	String() string
}

// NewIDFromHex creates a new ID that is validated.
func NewIDFromHex(value string) (ID, error) {
	return parseObjectIDFromHex(value)
}

// Ident is a git user identifier. These typically represent authors and committers.
type Ident interface {
	// Name is the name of the user.
	Name() string
	// Email is the email of the user.
	Email() string
	// Timestamp is the time at which this identity was created. For authors it's the
	// commit's author time, and for committers it's the commit's commit time.
	Timestamp() time.Time
}

// Commit represents a commit object.
//
// All commits will have a non-nil Tree. All but the root commit will contain >0 parents.
type Commit interface {
	// ID is the ID for this commit.
	ID() ID
	// Tree is the ID to the git tree for this commit.
	Tree() ID
	// Parents is the set of parents for this commit. It may be empty.
	//
	// By convention, the first parent in a multi-parent commit is the merge target.
	Parents() []ID
	// Author is the user who authored the commit.
	Author() Ident
	// Committer is the user who created the commit.
	Committer() Ident
	// Message is the commit message.
	Message() string
}

// Reader reads objects (commits, trees, blobs) from a `.git` directory.
type Reader interface {
	// Commit reads the commit identified by the ID.
	Commit(id ID) (Commit, error)
}

// NewReader creates a new Reader that can read objects from a `.git` directory.
//
// The provided path to the `.git` dir need not be normalized or cleaned.
//
// Internally, NewReader will spawns a new process to communicate with `git-cat-file`,
// so the caller must invoke the close function to clean up resources.
func NewReader(
	gitDirPath string,
	runner command.Runner,
) (Reader, func() error, error) {
	reader, err := newReader(gitDirPath, runner)
	return reader, reader.Close, err
}
