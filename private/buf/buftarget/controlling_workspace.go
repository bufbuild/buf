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
	// Can be a v1 or v2 buf.yaml.
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
