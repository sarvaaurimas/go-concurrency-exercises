package concurrency

import (
	"context"
	"sync"
	"time"
)

// =============================================================================
// EXERCISE 3.5: Semaphore Pattern
// =============================================================================
//
// A semaphore limits concurrent access to a resource.
// It's like a mutex but allows N concurrent accessors instead of just 1.
//
// KEY CONCEPTS:
// - Semaphore with count N allows N concurrent operations
// - Acquire() blocks until a slot is available
// - Release() frees a slot for others
// - Can be implemented with a buffered channel!
// - golang.org/x/sync/semaphore provides weighted semaphore
//
// =============================================================================

// =============================================================================
// PART 1: Channel-Based Semaphore
// =============================================================================

// Semaphore limits concurrent access using a buffered channel.
//
// TODO: Implement using a buffered channel of struct{}
// - Channel capacity = max concurrent operations
// - Acquire = send to channel (blocks when full)
// - Release = receive from channel
//
// QUESTION: Why use struct{} instead of bool or int?
type Semaphore struct {
	sem chan struct{}
}

// NewSemaphore creates a semaphore allowing n concurrent operations.
func NewSemaphore(n int) *Semaphore {
	// YOUR CODE HERE
	return &Semaphore{
		sem: make(chan struct{}, n),
	}
}

// Acquire blocks until a slot is available.
func (s *Semaphore) Acquire() {
	<-s.sem
}

// Release frees a slot.
func (s *Semaphore) Release() {
	s.sem <- struct{}{}
}

// TryAcquire attempts to acquire without blocking.
// Returns true if successful.
func (s *Semaphore) TryAcquire() bool {
	select {
	case <-s.sem:
		return true
	default:
		return false
	}
}

// =============================================================================
// PART 2: Semaphore with Timeout
// =============================================================================

// TimeoutSemaphore supports timeout on acquire.
type TimeoutSemaphore struct {
	sem chan struct{}
}

// NewTimeoutSemaphore creates a semaphore with timeout support.
func NewTimeoutSemaphore(n int) *TimeoutSemaphore {
	sem := make(chan struct{}, n)
	for range n {
		sem <- struct{}{}
	}
	return &TimeoutSemaphore{
		sem: sem,
	}
}

// Acquire blocks until a slot is available.
func (s *TimeoutSemaphore) Acquire() {
	<-s.sem
}

// AcquireTimeout tries to acquire within timeout.
// Returns true if acquired, false if timeout.
func (s *TimeoutSemaphore) AcquireTimeout(timeout time.Duration) bool {
	select {
	case <-s.sem:
		return true
	case <-time.After(timeout):
		return false
	}
}

// AcquireContext tries to acquire, respecting context cancellation.
func (s *TimeoutSemaphore) AcquireContext(ctx context.Context) error {
	select {
	case <-s.sem:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release frees a slot.
func (s *TimeoutSemaphore) Release() {
	s.sem <- struct{}{}
}

// =============================================================================
// PART 3: Bounded Parallel Execution
// =============================================================================

// BoundedParallel runs functions with limited concurrency.
//
// TODO: Implement to:
// 1. Run at most 'limit' functions concurrently
// 2. Return when all functions complete
// 3. Collect all results in order
//
// HINT: Use semaphore + WaitGroup
func BoundedParallel(fns []func() int, limit int) []int {
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	results := make([]int, len(fns))
	sem := NewSemaphore(limit)
	for i, fn := range fns {
		wg.Go(func() {
			sem.Acquire()
			res := fn()
			sem.Release()
			mu.Lock()
			// This keeps results in the same order
			results[i] = res
			mu.Unlock()
		})
	}
	wg.Wait()
	return results
}

// BoundedParallelError is like above but functions can return errors.
// If any function errors, return immediately (cancel pending work).
//
// TODO: Implement with early termination on error
// HINT: Combine semaphore, WaitGroup, context, and error channel
func BoundedParallelError(ctx context.Context, fns []func(context.Context) error, limit int) error {
	var wg sync.WaitGroup

	errs := make(chan error, len(fns))
	sem := NewTimeoutSemaphore(limit)
	localCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for _, fn := range fns {
			// If parent ctx is cancelled it will stop and return
			err := sem.AcquireContext(localCtx)
			if err != nil {
				// Dont spawn no more
				break
			}
			wg.Go(func() {
				defer sem.Release()
				errs <- fn(localCtx)
			})
		}
	}()
	go func() {
		wg.Wait()
		close(errs)
	}()

	for err := range errs {
		if err != nil {
			cancel()
			return err
		}
	}
	return nil
}

// =============================================================================
// PART 4: Weighted Semaphore
// =============================================================================

// WeightedSemaphore allows operations to acquire multiple slots.
// Useful when different operations need different amounts of a resource.
//
// TODO: Implement to:
// - Acquire(n) acquires n slots
// - Release(n) releases n slots
// - Total available slots is 'capacity'
//
// QUESTION: Why can't this be implemented with a simple buffered channel?

// We implement a semaphore where big acquisitions can be starved
type WeightedSemaphore struct {
	newAv  *sync.Cond
	currAv int64
}

// NewWeightedSemaphore creates a semaphore with given total capacity.
func NewWeightedSemaphore(capacity int64) *WeightedSemaphore {
	// YOUR CODE HERE
	return &WeightedSemaphore{
		currAv: capacity,
		newAv:  sync.NewCond(&sync.Mutex{}),
	}
}

// Acquire blocks until n slots are available.
func (ws *WeightedSemaphore) Acquire(n int64) {
	ws.newAv.L.Lock()
	defer ws.newAv.L.Unlock()
	// Wait if current available less than n
	for n > ws.currAv {
		ws.newAv.Wait()
	}
	ws.currAv -= n
}

// TryAcquire attempts to acquire n slots without blocking.
func (ws *WeightedSemaphore) TryAcquire(n int64) bool {
	ws.newAv.L.Lock()
	defer ws.newAv.L.Unlock()
	if n > ws.currAv {
		return false
	}
	ws.currAv -= n
	return true
}

// Release frees n slots.
func (ws *WeightedSemaphore) Release(n int64) {
	ws.newAv.L.Lock()
	defer ws.newAv.L.Unlock()
	ws.currAv += n
	// Wake up all and let them figure out if they have to go back to sleep or not
	ws.newAv.Broadcast()
}

// =============================================================================
// PART 5: Resource Pool
// =============================================================================

// Connection represents a pooled resource.
type Connection struct {
	ID int
}

// ConnectionPool manages a fixed pool of connections.
// Similar to database connection pools.
//
// TODO: Implement to:
// - Pool has N pre-created connections
// - Get() returns an available connection (blocks if none available)
// - Put() returns a connection to the pool
// - Close() closes all connections
//
// HINT: This is basically a semaphore + a slice of resources
type ConnectionPool struct {
	sem   chan struct{}
	mu    sync.Mutex
	conns []*Connection
}

// NewConnectionPool creates a pool with n connections.
func NewConnectionPool(n int, factory func(id int) *Connection) *ConnectionPool {
	conns := make([]*Connection, 0, n)
	sem := make(chan struct{}, n)
	for ID := range n {
		conns = append(conns, factory(ID))
		sem <- struct{}{}
	}
	return &ConnectionPool{
		sem:   sem,
		conns: conns,
	}
}

// Get retrieves a connection from the pool, blocking if necessary.
func (cp *ConnectionPool) Get() *Connection {
	<-cp.sem
	cp.mu.Lock()
	defer cp.mu.Unlock()
	lastidx := len(cp.conns) - 1
	c := cp.conns[lastidx]
	cp.conns = cp.conns[:lastidx]
	return c
}

// Put returns a connection to the pool.
func (cp *ConnectionPool) Put(conn *Connection) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.conns = append(cp.conns, conn)
	cp.sem <- struct{}{}
}

// Close closes all connections in the pool.
func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	// Can call close  if connections had a close func
	for i := range cp.conns {
		cp.conns[i] = nil
	}
}

// =============================================================================
// CHALLENGE: Implement a fair semaphore
// =============================================================================

// FairSemaphore ensures waiters are served in FIFO order.
// The basic channel semaphore doesn't guarantee ordering!
//
// TODO: Implement fair acquisition order
// HINT: Queue waiters and wake them in order
type FairSemaphore struct {
	// YOUR FIELDS HERE
}

// NewFairSemaphore creates a fair semaphore.
// func NewFairSemaphore(n int) *FairSemaphore {
// 	// YOUR CODE HERE
// 	return nil
// }

// Ensure imports are used
var (
	_ = context.Background
	_ = time.Second
)
