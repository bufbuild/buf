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

package bandeps

import (
	"crypto/sha256"
)

type violation struct {
	pkg  string
	dep  string
	note string
}

func newViolation(
	pkg string,
	dep string,
	note string,
) *violation {
	return &violation{
		pkg:  pkg,
		dep:  dep,
		note: note,
	}
}

func (v *violation) Package() string {
	return v.pkg
}

func (v *violation) Dep() string {
	return v.dep
}

func (v *violation) Note() string {
	return v.note
}

func (v *violation) String() string {
	return v.pkg + ` cannot depend on ` + v.dep + `: ` + v.note + `.`
}

func (v *violation) key() string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(v.pkg))
	_, _ = hash.Write([]byte(v.dep))
	_, _ = hash.Write([]byte(v.note))
	return string(hash.Sum(nil))
}
