package bytepool

import (
	"sync"
)

type pool struct {
	listStatsManager *listStatsManager
	size             uint32
	noMemsetZero     bool
	c                chan (*Bytes)
	syncPool         *sync.Pool
}

func newPool(listStatsManager *listStatsManager, chanSize uint32, size uint32, noMemsetZero bool) *pool {
	pool := &pool{
		listStatsManager: listStatsManager,
		size:             size,
		noMemsetZero:     noMemsetZero,
	}
	if chanSize > 0 {
		pool.c = make(chan *Bytes, chanSize)
	}
	pool.syncPool = &sync.Pool{
		New: func() interface{} {
			listStatsManager.RecordListNew(size)
			return newBytes(pool, int(size))
		},
	}
	return pool
}

func (p *pool) get() *Bytes {
	if p.c == nil {
		g := p.syncPool.Get().(*Bytes)
		p.afterGet(g)
		return g
	}
	select {
	case b := <-p.c:
		p.afterGet(b)
		return b
	default:
		g := p.syncPool.Get().(*Bytes)
		p.afterGet(g)
		return g
	}
}

func (p *pool) put(b *Bytes) {
	if p.c == nil {
		p.syncPool.Put(b)
		p.afterPut(b)
		return
	}
	select {
	case p.c <- b:
		p.afterPut(b)
	default:
		p.syncPool.Put(b)
		p.afterPut(b)
	}
}

func (p *pool) afterGet(b *Bytes) {
	b.reset()
	if !p.noMemsetZero {
		b.memsetZero()
	}
	p.listStatsManager.RecordListGet(p.size)
}

func (p *pool) afterPut(b *Bytes) {
	p.listStatsManager.RecordListRecycle(p.size)
}
