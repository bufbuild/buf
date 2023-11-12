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

package bufconfig

import (
	"errors"
	"io"
)

type genOnlyFile struct {
	generateConfig
}

func newGenOnlyFile() *genOnlyFile {
	return &genOnlyFile{}
}

func (g *genOnlyFile) FileVersion() FileVersion {
	panic("not implemented") // TODO: Implement
}

func (*genOnlyFile) isGenOnlyFile() {}

func readGenOnlyFile(reader io.Reader) (GenOnlyFile, error) {
	return nil, errors.New("TODO")
}

func writeGenOnlyFile(writer io.Writer, genOnlyFile GenOnlyFile) error {
	return errors.New("TODO")
}
