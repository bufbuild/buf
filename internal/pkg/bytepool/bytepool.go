// Package bytepool implements a pool of bytes using a seg list.
package bytepool

import (
	"io"
	"sort"
)

// DefaultPoolChanSize is the default pool channel size.
const DefaultPoolChanSize = uint32(32)

// DefaultListSizes are the default list sizes.
var DefaultListSizes = []uint32{
	8,
	16,
	64,
	256,
	512,
	1024,
	4096,
	8192,
	16384,
	32768,
	65536,
	131072,
	262144,
	524288,
	1048576,
	2097152,
	4194304,
	8388608,
	16777216,
}

// SegList is a seg list.
//
// Only create with NewSegList.
type SegList struct {
	poolChanSize     uint32
	noMemsetZero     bool
	listSizes        []uint32
	listSizeToPool   map[uint32]*pool
	listStatsManager *listStatsManager
}

// SegListOption is an option for a new SegList
type SegListOption func(*SegList)

// SegListWithPoolChanSize returns a SegListOption that uses the given pool channel size.
//
// The default is to use DefaultPoolChanSize.
//
// If poolChanSize is 0, no channel will be used and all allocations will be done
// with a sync.Pool.
func SegListWithPoolChanSize(poolChanSize uint32) SegListOption {
	return func(segList *SegList) {
		segList.poolChanSize = poolChanSize
	}
}

// SegListWithListSizes returns a SegListOption that uses the given list sizes.
//
// The default is to use DefaultListSizes.
//
// If listSizes is nil or empty, no lists will be used and all allocations will
// be done with make.
func SegListWithListSizes(listSizes []uint32) SegListOption {
	return func(segList *SegList) {
		segList.listSizes = listSizes
	}
}

// SegListWithNoMemsetZero returns a SegListOption that does not clear allocated buffers.
//
// Only do this if you know what you are doing.
func SegListWithNoMemsetZero() SegListOption {
	return func(segList *SegList) {
		segList.noMemsetZero = true
	}
}

// NewNoPoolSegList returns a new SegList that does no memory pooling.
func NewNoPoolSegList() *SegList {
	return NewSegList(
		SegListWithPoolChanSize(0),
		SegListWithListSizes(nil),
	)
}

// NewSegList returns a new SegList.
func NewSegList(options ...SegListOption) *SegList {
	segList := &SegList{
		poolChanSize: DefaultPoolChanSize,
		listSizes:    DefaultListSizes,
		noMemsetZero: false,
	}
	for _, option := range options {
		option(segList)
	}

	if len(segList.listSizes) > 0 {
		listSizesCopy := make([]uint32, 0, len(segList.listSizes))
		for _, listSize := range segList.listSizes {
			if listSize != 0 {
				listSizesCopy = append(listSizesCopy, listSize)
			}
		}
		if len(listSizesCopy) > 0 {
			sort.Sort(uint32Slice(listSizesCopy))
		}
		segList.listSizes = listSizesCopy
	}

	segList.listStatsManager = newListStatsManager(segList.listSizes)
	segList.listSizeToPool = make(map[uint32]*pool, len(segList.listSizes))
	for _, listSize := range segList.listSizes {
		segList.listSizeToPool[listSize] = newPool(
			segList.listStatsManager,
			segList.poolChanSize,
			listSize,
			segList.noMemsetZero,
		)
	}
	return segList
}

// Get gets a Bytes of at least the given size.
//
// Returns nil if size == 0.
func (s *SegList) Get(size uint32) *Bytes {
	if size == 0 {
		return nil
	}
	if len(s.listSizes) == 0 {
		s.listStatsManager.RecordListNew(0)
		return newBytes(nil, int(size))
	}
	// s.listSizes is sorted
	for _, listSize := range s.listSizes {
		if size <= listSize {
			pool, ok := s.listSizeToPool[listSize]
			if !ok {
				// this should never happen if NewSegList was done correctly
				panic("no pool of size")
			}
			return pool.get()
		}
	}
	s.listStatsManager.RecordListNew(0)
	return newBytes(nil, int(size))
}

// ListStats returns the list stats.
func (s *SegList) ListStats() []*ListStats {
	return s.listStatsManager.ListStats()
}

// ListStats are stats.
type ListStats struct {
	// The list size.
	// 0 denotes no list.
	ListSize uint32
	// Number of times New was called on the sync.Pool.
	TotalNew uint64
	// Number of times Get was called on the *Pool.
	TotalGet uint64
	// Number of outstanding messages that were not recycled.
	TotalUnrecycled uint64
}

// Bytes represents a byte slice.
//
// Only create these from SegLists.
type Bytes struct {
	data   []byte
	pool   *pool
	curLen int
	dirty  bool
}

// CopyFrom copies from the byte slice to the Bytes starting at the offset.
//
// Returns io.EOF if len(from) + offset is greater than the buffer size.
func (b *Bytes) CopyFrom(from []byte, offset int) (int, error) {
	if b.dirty {
		panic("use after free")
	}
	end := len(from) + offset
	if end > len(b.data) {
		return 0, io.EOF
	}
	copy(b.data[offset:end], from)
	if b.curLen < end {
		b.curLen = end
	}
	return len(from), nil
}

// CopyTo copies the from the Bytes to the byte slice starting at the offset.
//
// Returns io.EOF if len(to) + offset is greater than Len().
func (b *Bytes) CopyTo(to []byte, offset int) (int, error) {
	if b.dirty {
		panic("use after free")
	}
	end := len(to) + offset
	if end > b.curLen {
		return 0, io.EOF
	}
	copy(to, b.data[offset:end])
	return len(to), nil
}

// Len gets the current length.
func (b *Bytes) Len() int {
	if b.dirty {
		panic("use after free")
	}
	return b.curLen
}

// Recycle recycles the Bytes.
//
// Must be called when done.
func (b *Bytes) Recycle() {
	if b.dirty {
		panic("double free")
	}
	b.dirty = true
	if b.pool != nil {
		b.pool.put(b)
	}
}

// memsetZero sets all the data to zero.
//
// Does nothing to the current length.
//
// Optimized per https://golang.org/cl/137880043
func (b *Bytes) memsetZero() {
	for i := range b.data {
		b.data[i] = 0
	}
}

// reset sets the current length to 0 and clears the dirty bit.
//
// Does nothing to the underlying byte slice.
func (b *Bytes) reset() {
	b.curLen = 0
	b.dirty = false
}

func newBytes(pool *pool, size int) *Bytes {
	return &Bytes{
		data:   make([]byte, size),
		pool:   pool,
		curLen: 0,
		dirty:  false,
	}
}

type uint32Slice []uint32

func (p uint32Slice) Len() int               { return len(p) }
func (p uint32Slice) Less(i int, j int) bool { return p[i] < p[j] }
func (p uint32Slice) Swap(i int, j int)      { p[i], p[j] = p[j], p[i] }
