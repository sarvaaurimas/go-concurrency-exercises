package concurrency

import (
	"sync"
)

// =============================================================================
// EXERCISE 2.3: sync.Once and sync.Cond
// =============================================================================
//
// sync.Once ensures a function is only executed once, even across goroutines.
// sync.Cond provides a way to wait for and signal conditions.
//
// KEY CONCEPTS:
// - Once.Do(f) calls f only the first time, subsequent calls are no-ops
// - Once is useful for lazy initialization of singletons
// - Cond.Wait() releases the lock, waits for signal, re-acquires lock
// - Cond.Signal() wakes ONE waiting goroutine
// - Cond.Broadcast() wakes ALL waiting goroutines
// - ALWAYS check condition in a loop after Wait() returns!
//
// GO 1.21+ ADDITIONS:
// - sync.OnceFunc(f) returns a function that calls f once (handles panics better)
// - sync.OnceValue[T](f) returns a function that calls f once and returns value
// - sync.OnceValues[T1,T2](f) returns a function that returns two values
//
// =============================================================================

// =============================================================================
// PART 1: sync.Once - Lazy Initialization
// =============================================================================

// ExpensiveResource simulates a resource that's expensive to create.
type ExpensiveResource struct {
	data string
}

// ResourceManager provides lazy-initialized access to ExpensiveResource.
//
// TODO: Implement lazy initialization using sync.Once
// - The resource should only be created when first requested
// - Multiple concurrent calls to Get should all receive the same instance
// - createFn should only be called once, ever
type ResourceManager struct {
	once     sync.Once
	createFn func() *ExpensiveResource
	res      *ExpensiveResource
}

// NewResourceManager creates a manager with the given creation function.
func NewResourceManager(createFn func() *ExpensiveResource) *ResourceManager {
	// YOUR CODE HERE
	return &ResourceManager{
		createFn: createFn,
	}
}

// Get returns the resource, creating it if necessary.
func (rm *ResourceManager) Get() *ExpensiveResource {
	rm.once.Do(func() {
		rm.res = rm.createFn()
	})
	return rm.res
}

// =============================================================================
// PART 2: sync.Once - Configuration Loading
// =============================================================================

// Config represents application configuration.
type Config struct {
	DatabaseURL string
	APIKey      string
	Debug       bool
}

// ConfigLoader loads configuration exactly once.
//
// TODO: Implement so that:
// - Load() reads config (calls loader function)
// - Load() is safe to call from multiple goroutines
// - The loader function is only called once
// - Get() returns the loaded config (nil if not loaded)
// - Get() is non-blocking
//
// QUESTION: What happens if the loader function panics?
// ANSWER: sync.Once records that Do was called even if f panics! Subsequent
// calls to Do will NOT retry the function. If initialization can fail, you
// may need sync.OnceFunc (Go 1.21+) or a custom pattern with OnceValue/OnceValues.
type ConfigLoader struct {
	// YOUR FIELDS HERE
	getConfig func() *Config
}

// NewConfigLoader creates a loader with the given loading function.
func NewConfigLoader(loader func() *Config) *ConfigLoader {
	// YOUR CODE HERE
	return &ConfigLoader{
		getConfig: sync.OnceValue(loader),
	}
}

// Get returns the loaded config, or nil if not yet loaded.
func (cl *ConfigLoader) Get() *Config {
	return cl.getConfig()
}

// =============================================================================
// PART 3: sync.Cond - Producer/Consumer
// =============================================================================

// BoundedQueue is a thread-safe queue with a maximum size.
// Producers block when full, consumers block when empty.
//
// TODO: Implement using sync.Cond
// - Put blocks if queue is full
// - Get blocks if queue is empty
// - Signal waiting goroutines appropriately
//
// QUESTION: Why use Cond instead of channels here?
type BoundedQueue struct {
	mu       sync.Mutex
	notFull  *sync.Cond
	notEmpty *sync.Cond
	queue    []int
}

// NewBoundedQueue creates a queue with the given capacity.
func NewBoundedQueue(capacity int) *BoundedQueue {
	q := &BoundedQueue{
		queue: make([]int, 0, capacity),
	}
	q.notFull = sync.NewCond(&q.mu)
	q.notEmpty = sync.NewCond(&q.mu)
	return q
}

// Put adds an item to the queue, blocking if full.
func (q *BoundedQueue) Put(item int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Wait if full
	for len(q.queue) == cap(q.queue) {
		q.notFull.Wait()
	}
	q.queue = append(q.queue, item)
	// Wake one sleeping get
	q.notEmpty.Signal()
}

// Get removes and returns an item, blocking if empty.
func (q *BoundedQueue) Get() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Wait if empty
	for len(q.queue) == 0 {
		q.notEmpty.Wait()
	}
	item := q.queue[0]
	q.queue = q.queue[1:]
	// Wake one sleeping put
	q.notFull.Signal()
	return item
}

// Len returns current number of items in queue.
func (q *BoundedQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue)
}

// =============================================================================
// PART 4: sync.Cond - Barrier
// =============================================================================

// Barrier blocks goroutines until N have arrived, then releases all.
// This is useful for synchronizing phases of computation.
//
// TODO: Implement using sync.Cond
// - Wait() blocks until n goroutines have called Wait()
// - Once n goroutines are waiting, all are released
// - Barrier can be reused for multiple rounds
//
// QUESTION: How is this different from WaitGroup?
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

// =============================================================================
// CHALLENGE: Implement a read-write barrier
// =============================================================================

// RWBarrier allows writers to proceed only when all readers are done.
//
// TODO: Implement so that:
// - StartRead() marks a reader as active
// - EndRead() marks a reader as done
// - WaitForReaders() blocks until all active readers are done
// - Multiple writers can wait simultaneously
//
// HINT: Track count of active readers, use Broadcast when count hits 0
type RWBarrier struct {
	// YOUR FIELDS HERE
}

// NewRWBarrier creates a new read-write barrier.
// func NewRWBarrier() *RWBarrier {
// 	// YOUR CODE HERE
// 	return nil
// }

// Ensure sync import is used
var (
	_ = sync.Once{}
	_ = sync.Cond{}
)
