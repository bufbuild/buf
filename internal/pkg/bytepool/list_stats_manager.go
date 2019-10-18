package bytepool

import (
	"sync"
)

type listStatsManager struct {
	sizes           []uint32
	sizeToListStats map[uint32]*listStatsWrapper
}

func newListStatsManager(sizes []uint32) *listStatsManager {
	listStatsManager := &listStatsManager{
		sizes:           sizes,
		sizeToListStats: make(map[uint32]*listStatsWrapper, len(sizes)+1),
	}
	for _, size := range sizes {
		listStatsManager.sizeToListStats[size] = &listStatsWrapper{listStats: &ListStats{ListSize: size}}
	}
	listStatsManager.sizeToListStats[0] = &listStatsWrapper{listStats: &ListStats{ListSize: 0}}
	return listStatsManager
}

func (m *listStatsManager) ListStats() []*ListStats {
	s := make([]*ListStats, 0, len(m.sizes))
	// guarantees lock ordering
	for _, size := range m.sizes {
		listStatsWrapper := m.sizeToListStats[size]
		listStatsWrapper.RLock()
		s = append(s, listStatsWrapper.listStats)
		listStatsWrapper.RUnlock()
	}
	return s
}

func (m *listStatsManager) RecordListNew(size uint32) {
	listStatsWrapper := m.sizeToListStats[size]
	listStatsWrapper.Lock()
	listStatsWrapper.listStats.TotalNew++
	listStatsWrapper.Unlock()
}

func (m *listStatsManager) RecordListGet(size uint32) {
	listStatsWrapper := m.sizeToListStats[size]
	listStatsWrapper.Lock()
	listStatsWrapper.listStats.TotalGet++
	listStatsWrapper.listStats.TotalUnrecycled++
	listStatsWrapper.Unlock()
}

func (m *listStatsManager) RecordListRecycle(size uint32) {
	listStatsWrapper := m.sizeToListStats[size]
	listStatsWrapper.Lock()
	listStatsWrapper.listStats.TotalUnrecycled--
	listStatsWrapper.Unlock()
}

type listStatsWrapper struct {
	sync.RWMutex
	listStats *ListStats
}
