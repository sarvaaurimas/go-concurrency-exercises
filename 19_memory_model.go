package concurrency

import (
	"sync"
	"sync/atomic"
)

// =============================================================================
// EXERCISE 5.4: Go Memory Model and Happens-Before
// =============================================================================
//
// The Go memory model specifies when writes in one goroutine are visible
// to reads in another. Understanding this prevents subtle bugs.
//
// KEY CONCEPTS:
// - "Happens-before" is a partial ordering of memory operations
// - If A happens-before B, A's effects are visible to B
// - Without synchronization, there's NO guarantee about visibility
// - Synchronization primitives establish happens-before relationships
//
// HAPPENS-BEFORE RULES:
// 1. Within a goroutine: statements happen in program order
// 2. Channel send happens-before corresponding receive completes
// 3. Channel close happens-before receive that returns zero value
// 4. Receive from unbuffered channel happens-before send completes
// 5. Lock() happens-before any Unlock() that follows
// 6. sync.Once Do() happens-before any Do() returns
// 7. Atomic operations provide sequential consistency
//
// =============================================================================

// =============================================================================
// PART 1: Visibility Without Synchronization
// =============================================================================

// UnsafeVisibility demonstrates that without sync, changes may not be visible.
//
// QUESTION: What might secondGoroutineSaw return? (0? 42? something else?)
// ANSWER: ANY of these! Without synchronization, there's no guarantee.
func UnsafeVisibility() int {
	var value int
	var seen int

	go func() {
		value = 42
	}()

	go func() {
		seen = value // May see 0 or 42 - undefined!
	}()

	// Don't do this - just for demonstration
	// time.Sleep(time.Millisecond)

	return seen // This is also racy!
}

// ChannelVisibility demonstrates happens-before with channels.
//
// TODO: Fix UnsafeVisibility using a channel to establish happens-before.
// The second goroutine should ALWAYS see value = 42.
func ChannelVisibility() int {
	// YOUR CODE HERE
	return 0
}

// MutexVisibility demonstrates happens-before with mutex.
//
// TODO: Fix UnsafeVisibility using mutex.
func MutexVisibility() int {
	// YOUR CODE HERE
	return 0
}

// =============================================================================
// PART 2: The "Happens-Before" Relationship
// =============================================================================

// Understanding which operations establish happens-before.

// ChannelHappensBefore demonstrates the channel happens-before rules.
//
// Rule: A send on a channel happens-before the corresponding receive completes.
//
// TODO: Explain why this code is correct (goroutine 2 always sees a=1, b=2):
func ChannelHappensBefore() (int, int) {
	var a, b int
	ch := make(chan struct{})

	go func() {
		a = 1
		b = 2
		ch <- struct{}{} // Send happens-before...
	}()

	<-ch // ...receive completes
	return a, b // Guaranteed to be 1, 2
}

// UnbufferedChannelOrder demonstrates receive-before-send-completes rule.
//
// Rule: For unbuffered channels, receive happens-before send COMPLETES.
//
// TODO: Explain why this is correct:
func UnbufferedChannelOrder() int {
	var value int
	ch := make(chan struct{})

	go func() {
		value = 42
		<-ch // Receive happens first...
	}()

	ch <- struct{}{} // ...then send completes
	return value // Guaranteed to be 42
}

// BufferedChannelOrder shows that buffered channels are different.
//
// QUESTION: Is this code correct? Why or why not?
func BufferedChannelOrder() int {
	var value int
	ch := make(chan struct{}, 1)

	go func() {
		value = 42
		ch <- struct{}{} // Doesn't wait for receiver
	}()

	<-ch
	return value // Is this guaranteed to be 42?
}

// =============================================================================
// PART 3: Atomic Operations and Memory Ordering
// =============================================================================

// AtomicVisibility demonstrates atomic operations provide visibility.
//
// Atomic operations create a total order that all goroutines agree on.
//
// TODO: Fix using atomic operations
func AtomicVisibility() int64 {
	var value int64
	var ready int64

	go func() {
		atomic.StoreInt64(&value, 42)
		atomic.StoreInt64(&ready, 1)
	}()

	// Spin until ready (don't do this in real code!)
	for atomic.LoadInt64(&ready) == 0 {
		// busy wait
	}

	return atomic.LoadInt64(&value) // Guaranteed to be 42
}

// AtomicWrongUsage shows a common mistake.
//
// QUESTION: What's wrong with this code?
func AtomicWrongUsage() int64 {
	var value int64
	var ready int64

	go func() {
		value = 42 // Non-atomic write!
		atomic.StoreInt64(&ready, 1)
	}()

	for atomic.LoadInt64(&ready) == 0 {
	}

	return value // Non-atomic read - is this safe?
}

// AtomicFix fixes the above code.
//
// TODO: Make both accesses atomic
func AtomicFix() int64 {
	// YOUR CODE HERE
	return 0
}

// =============================================================================
// PART 4: sync.Once Guarantees
// =============================================================================

// OnceHappensBefore demonstrates sync.Once provides happens-before.
//
// Rule: The function passed to once.Do() happens-before any Do() returns.
//
// TODO: Implement singleton initialization using sync.Once
// Multiple goroutines should all see the fully initialized value.
type ExpensiveSingleton struct {
	data []int
}

var (
	singletonInstance *ExpensiveSingleton
	singletonOnce     sync.Once
)

func GetSingletonInstance() *ExpensiveSingleton {
	// YOUR CODE HERE
	return nil
}

// =============================================================================
// PART 5: Publication Safety
// =============================================================================

// SafePublication demonstrates how to safely publish an object.
// "Publication" means making a reference visible to other goroutines.
//
// WRONG: Just assigning the pointer (no happens-before)
// RIGHT: Use channel, mutex, or atomic to establish happens-before

type AppConfig struct {
	Servers []string
	Timeout int
}

// UnsafePublication demonstrates incorrect publication.
// Other goroutines might see a partially constructed AppConfig!
var unsafeConfig *AppConfig

func UnsafePublish() {
	cfg := &AppConfig{
		Servers: []string{"a", "b", "c"},
		Timeout: 30,
	}
	unsafeConfig = cfg // WRONG: no synchronization
}

// SafePublicationAtomic uses atomic.Value for safe publication.
//
// TODO: Implement safe publication using atomic.Value
var safeAppConfig atomic.Value // Will hold *AppConfig

func SafePublish() {
	// YOUR CODE HERE
}

func GetAppConfig() *AppConfig {
	// YOUR CODE HERE
	return nil
}

// =============================================================================
// PART 6: Practical Examples
// =============================================================================

// LazyInitMap demonstrates safe lazy initialization of a map.
//
// TODO: Implement a map that initializes lazily in a thread-safe way
// Use sync.RWMutex with double-check locking pattern
type LazyMap struct {
	// YOUR FIELDS HERE
}

func NewLazyMap() *LazyMap {
	// YOUR CODE HERE
	return nil
}

func (m *LazyMap) GetOrCreate(key string, create func() interface{}) interface{} {
	// YOUR CODE HERE
	return nil
}

// =============================================================================
// CHALLENGE: Implement a lock-free stack
// =============================================================================

// AtomicStack implements a stack using atomic compare-and-swap.
// This is an advanced topic - understand happens-before first!
//
// TODO: Implement using atomic.CompareAndSwapPointer (or atomic.Value)
// 1. Push adds to top of stack
// 2. Pop removes and returns top
// 3. No locks allowed!
//
// HINT: Each operation reads current head, prepares new state,
// then atomically swaps if head hasn't changed.
type AtomicStackNode struct {
	value interface{}
	next  *AtomicStackNode
}

type AtomicStack struct {
	// YOUR FIELDS HERE
}

func NewAtomicStack() *AtomicStack {
	// YOUR CODE HERE
	return nil
}

func (s *AtomicStack) Push(value interface{}) {
	// YOUR CODE HERE
}

func (s *AtomicStack) Pop() (interface{}, bool) {
	// YOUR CODE HERE
	return nil, false
}

// Ensure imports are used
var _ = sync.Mutex{}
var _ = atomic.Value{}
