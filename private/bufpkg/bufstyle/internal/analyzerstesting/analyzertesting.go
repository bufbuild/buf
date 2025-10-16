package analyzerstesting

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// Run runs the tests.
//
// It expects tests to be in "testdata/src/p" relative to the Go package the tests are being run in.
func Run(t *testing.T, analyzer *analysis.Analyzer) {
	pwd, err := os.Getwd()
	require.NoError(t, err)
	analysistest.Run(t, filepath.Join(pwd, "testdata"), analyzer, "p")
}
