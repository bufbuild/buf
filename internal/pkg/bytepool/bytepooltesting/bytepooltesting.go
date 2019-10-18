// Package bytepooltesting implements testing functionality for bytepool.
package bytepooltesting

import (
	"testing"

	"github.com/bufbuild/buf/internal/pkg/bytepool"
	"github.com/stretchr/testify/assert"
)

// AssertAllRecycled asserts that all Bytes were recycled.
func AssertAllRecycled(t *testing.T, segList *bytepool.SegList) {
	var unrecycled uint64
	for _, listStats := range segList.ListStats() {
		unrecycled += listStats.TotalUnrecycled
	}
	assert.True(t, unrecycled == 0)
}
