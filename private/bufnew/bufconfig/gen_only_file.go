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

const (
	// DefaultGenOnlyFileName is the default file name you should use for buf.gen.yaml Files.
	//
	// This is not included in AllFileNames.
	//
	// For v2, generation configuration is merged into buf.yaml.
	DefaultGenOnlyFileName = "buf.gen.yaml"
)

// GenOnlyFile represents a buf.gen.yaml file.
//
// For v2, generation configuration has been merged into Files.
type GenOnlyFile interface {
	GenerateConfig

	// FileVersion returns the version of the buf.gen.yaml file this was read from.
	FileVersion() FileVersion

	isGenOnlyFile()
}

// ReadGenOnlyFile reads the GenOnlyFile from the io.Reader.
func ReadGenOnlyFile(reader io.Reader) (GenOnlyFile, error) {
	genOnlyFile, err := readGenOnlyFile(reader)
	if err != nil {
		return nil, err
	}
	if err := checkV2SupportedYet(genOnlyFile.FileVersion()); err != nil {
		return nil, err
	}
	return genOnlyFile, nil
}

// WriteGenOnlyFile writes the GenOnlyFile to the io.Writer.
func WriteGenOnlyFile(writer io.Writer, genOnlyFile GenOnlyFile) error {
	if err := checkV2SupportedYet(genOnlyFile.FileVersion()); err != nil {
		return err
	}
	return writeGenOnlyFile(writer, genOnlyFile)
}

// *** PRIVATE ***

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
