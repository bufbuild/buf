package bufmodule

const (
	// licenseFilePath is the path of the license file within a Module.
	licenseFilePath = "LICENSE"
)

var (
	// orderedDocFilePaths are the potential documentation file paths for a Module.
	//
	// When creating a Module from a Bucket, we check the file paths buf.md, README.md, and README.markdown
	// to exist, in that order. The first one to exist is chosen as the documentation file that is considered
	// part of the Module, and any others are discarded.
	orderedDocFilePaths = []string{
		"buf.md",
		"README.md",
		"README.markdown",
	}
)
