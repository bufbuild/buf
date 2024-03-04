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

package buftarget

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
)

type ControllingWorkspace interface {
	// Path of the controlling workspace in the bucket.
	Path() string
	// Returns a buf.work.yaml file that was found. This is empty if we are retruning a buf.yaml.
	BufWorkYAMLFile() bufconfig.BufWorkYAMLFile
	// Returns a buf.yaml that was found. This is empty if we are returning a buf.work.yaml.
	BufYAMLFile() bufconfig.BufYAMLFile
}

func NewControllingWorkspace(
	path string,
	bufWorkYAMLFile bufconfig.BufWorkYAMLFile,
	bufYAMLFile bufconfig.BufYAMLFile,
) ControllingWorkspace {
	return newControllingWorkspace(path, bufWorkYAMLFile, bufYAMLFile)
}

// *** PRIVATE ***

var (
	_ ControllingWorkspace = &controllingWorkspace{}
)

type controllingWorkspace struct {
	path            string
	bufWorkYAMLFile bufconfig.BufWorkYAMLFile
	bufYAMLFile     bufconfig.BufYAMLFile
}

func newControllingWorkspace(
	path string,
	bufWorkYAMLFile bufconfig.BufWorkYAMLFile,
	bufYAMLFile bufconfig.BufYAMLFile,
) ControllingWorkspace {
	return &controllingWorkspace{
		path:            path,
		bufWorkYAMLFile: bufWorkYAMLFile,
		bufYAMLFile:     bufYAMLFile,
	}
}

func (c *controllingWorkspace) Path() string {
	return c.path
}

func (c *controllingWorkspace) BufWorkYAMLFile() bufconfig.BufWorkYAMLFile {
	return c.bufWorkYAMLFile
}

func (c *controllingWorkspace) BufYAMLFile() bufconfig.BufYAMLFile {
	return c.bufYAMLFile
}
