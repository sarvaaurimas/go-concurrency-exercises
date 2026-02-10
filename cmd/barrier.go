package main

import "sync"

type Barrier struct {
	ready  *sync.Cond
	capGo  int
	currGo int
}

// NewBarrier creates a barrier for n goroutines.
func NewBarrier(n int) *Barrier {
	return &Barrier{
		ready: sync.NewCond(&sync.Mutex{}),
		capGo: n,
	}
}

// Wait blocks until all n goroutines have reached the barrier.
// Returns the order in which this goroutine arrived (1 to n).
func (b *Barrier) Wait() (order int) {
	b.ready.L.Lock()
	defer b.ready.L.Unlock()

	b.currGo++
	order = b.currGo
	if b.currGo < b.capGo {
		b.ready.Wait()
	} else {
		b.ready.Broadcast()
		b.currGo = 0
	}
	return order
}
