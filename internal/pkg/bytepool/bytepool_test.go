package bytepool

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This package is implicitly tested by the rest of Buf, but needs a *lot*
// more specific testing of it's functionality.

func TestBasic1(t *testing.T) {
	t.Parallel()

	// putting in the wrong order on purpose
	segList := NewSegList(SegListWithListSizes([]uint32{16, 8}))

	bytes1 := segList.Get(7)
	// make sure that bytes1 != bytes2
	bytes2 := segList.Get(8)
	bytes1.Recycle()
	bytes3 := segList.Get(16)
	bytes4 := segList.Get(32)
	// bytes1 should == bytes5
	bytes5 := segList.Get(8)
	assert.Equal(t, bytes1, bytes5)
	bytes5.Recycle()
	assert.Equal(
		t,
		[]*ListStats{
			&ListStats{
				ListSize:        8,
				TotalNew:        2,
				TotalGet:        3,
				TotalUnrecycled: 1,
			},
			&ListStats{
				ListSize:        16,
				TotalNew:        1,
				TotalGet:        1,
				TotalUnrecycled: 1,
			},
		},
		segList.ListStats(),
	)
	testBasicPanics(t, bytes1, bytes5)

	n, err := bytes3.CopyFrom(make([]byte, 8), 0)
	assert.NoError(t, err)
	assert.Equal(t, 8, n)
	n, err = bytes3.CopyFrom(make([]byte, 8), 9)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)

	bytes2.Recycle()
	bytes3.Recycle()
	bytes4.Recycle()
	assert.Equal(
		t,
		[]*ListStats{
			&ListStats{
				ListSize:        8,
				TotalNew:        2,
				TotalGet:        3,
				TotalUnrecycled: 0,
			},
			&ListStats{
				ListSize:        16,
				TotalNew:        1,
				TotalGet:        1,
				TotalUnrecycled: 0,
			},
		},
		segList.ListStats(),
	)

	testBasicPanics(t, bytes1, bytes2, bytes3, bytes4, bytes5)
}

func testBasicPanics(t *testing.T, allBytes ...*Bytes) {
	for _, bytes := range allBytes {
		assert.PanicsWithValue(t, "use after free", func() { _, _ = bytes.CopyFrom(make([]byte, 0), 0) })
		assert.PanicsWithValue(t, "use after free", func() { _, _ = bytes.CopyTo(make([]byte, 16), 0) })
		assert.PanicsWithValue(t, "use after free", func() { bytes.Len() })
		assert.PanicsWithValue(t, "use after free", func() { bytes.Len() })
		assert.PanicsWithValue(t, "double free", func() { bytes.Recycle() })
	}
}
