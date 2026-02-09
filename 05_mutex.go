package concurrency

import (
	"sync"
	"time"
)

// =============================================================================
// EXERCISE 2.2: sync.Mutex vs sync.RWMutex
// =============================================================================
//
// Mutexes provide mutual exclusion for protecting shared data.
//
// KEY CONCEPTS:
// - Mutex: only one goroutine can hold the lock at a time
// - RWMutex: multiple readers OR one writer (but not both)
// - Always unlock in the same goroutine that locked
// - Use defer mu.Unlock() to ensure unlock even on panic
// - Copying a mutex after first use is forbidden (pass by pointer!)
//
// =============================================================================

// =============================================================================
// PART 1: Basic Mutex - Thread-Safe Counter
// =============================================================================

// Counter is a thread-safe counter.
//
// TODO: Implement all methods to be goroutine-safe
type Counter struct {
	mu    sync.Mutex
	count int
}

// NewCounter creates a new counter starting at 0.
func NewCounter() *Counter {
	// YOUR CODE HERE
	return &Counter{}
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

// Dec decrements the counter by 1.
func (c *Counter) Dec() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count--
}

// Value returns the current count.
func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// =============================================================================
// PART 2: RWMutex - Thread-Safe Cache
// =============================================================================

// Cache is a thread-safe string cache optimized for read-heavy workloads.
//
// TODO: Implement using RWMutex
// - Reads should not block other reads
// - Writes should have exclusive access
//
// QUESTION: When would you use Mutex instead of RWMutex?
type Cache struct {
	mu      sync.RWMutex
	storage map[string]string
}

// NewCache creates a new empty cache.
func NewCache() *Cache {
	// YOUR CODE HERE
	return &Cache{
		storage: map[string]string{},
	}
}

// Get retrieves a value from the cache.
// Returns (value, true) if found, ("", false) if not.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, exists := c.storage[key]
	return val, exists
}

// Set stores a value in the cache.
func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storage[key] = value
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.storage, key)
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.storage)
}

// =============================================================================
// PART 3: Lock Ordering to Prevent Deadlocks
// =============================================================================

// Account represents a bank account.
type Account struct {
	ID      int
	balance int
	mu      sync.Mutex
}

// NewAccount creates an account with initial balance.
func NewAccount(id, balance int) *Account {
	return &Account{ID: id, balance: balance}
}

// Balance returns current balance.
func (a *Account) Balance() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.balance
}

func HigherID(from, to *Account) (first, second *Account) {
	if from.ID >= to.ID {
		return from, to
	}
	return to, from
}

// Transfer moves amount from account 'from' to account 'to'.
//
// TODO: Implement this WITHOUT deadlocks!
//
// WRONG approach (causes deadlock):
//
//	from.mu.Lock()
//	to.mu.Lock()    // If another goroutine does Transfer(to, from, x), deadlock!
//
// QUESTION: How do you prevent deadlock when locking multiple mutexes?
// HINT: Always acquire locks in a consistent order (e.g., by account ID)
func Transfer(from, to *Account, amount int) bool {
	if from == to {
		return true
	}

	// Always lock and unlock higherID account first to prevent deadlocks
	first, second := HigherID(from, to)
	first.mu.Lock()
	defer first.mu.Unlock()
	second.mu.Lock()
	defer second.mu.Unlock()

	if from.balance < amount {
		return false
	}

	from.balance -= amount
	to.balance += amount

	return false
}

// =============================================================================
// PART 4: Mutex Patterns
// =============================================================================

// SafeSlice is a thread-safe slice.
//
// TODO: Implement append, get, and len operations
type SafeSlice struct {
	// YOUR FIELDS HERE
}

// NewSafeSlice creates an empty thread-safe slice.
func NewSafeSlice() *SafeSlice {
	// YOUR CODE HERE
	return nil
}

// Append adds a value to the slice.
func (s *SafeSlice) Append(val int) {
	// YOUR CODE HERE
}

// Get returns value at index. Panics if out of bounds.
func (s *SafeSlice) Get(index int) int {
	// YOUR CODE HERE
	return 0
}

// Len returns the length of the slice.
func (s *SafeSlice) Len() int {
	// YOUR CODE HERE
	return 0
}

// =============================================================================
// CHALLENGE: Implement a Read-Write cache with expiration
// =============================================================================

// ExpiringCache stores values that expire after a duration.
//
// TODO: Implement a cache where:
// - Each entry has a timestamp
// - Get returns ("", false) for expired entries
// - Set stores value with current time
// - CleanExpired removes all expired entries
//
// HINT: You'll need time.Time for timestamps and time.Duration for expiry

type CacheEntry struct {
	val          string
	expTimestamp time.Time
}

type ExpiringCache struct {
	mu   sync.RWMutex
	vals map[string]CacheEntry
	TTL  time.Duration
}

func NewExpiringCache(ttl time.Duration) *ExpiringCache {
	return &ExpiringCache{
		vals: map[string]CacheEntry{},
		TTL:  ttl,
	}
}

func (c *ExpiringCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.vals[key]

	// If doesn't exist or already expired
	if !ok || entry.expTimestamp.Before(time.Now()) {
		return "", false
	}
	return entry.val, true
}

func (c *ExpiringCache) Set(key, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vals[key] = CacheEntry{
		val:          val,
		expTimestamp: time.Now().Add(c.TTL),
	}
}

func (c *ExpiringCache) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.vals {
		if v.expTimestamp.Before(now) {
			delete(c.vals, k)
		}
	}
}

// Ensure sync import is used
var _ = sync.Mutex{}
