package bufsync_test

import (
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/buf/bufsync/bufsynctest"
)

func TestSyncer(t *testing.T) {
	bufsynctest.RunTestSuite(t, func() bufsync.Handler {
		return newTestSyncHandler()
	})
}
