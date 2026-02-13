package concurrency

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// =============================================================================
// EXERCISE 5.3: singleflight for Request Deduplication
// =============================================================================
//
// singleflight (golang.org/x/sync/singleflight) ensures that only one
// execution of a function is in-flight at a time. Duplicate calls wait
// for the original to complete and share the result.
//
// KEY CONCEPTS:
// - Prevents "thundering herd" / cache stampede problems
// - Key-based deduplication (same key = same operation)
// - All callers get the same result (and error)
// - g.Do(key, fn) - execute fn, deduplicate by key
// - g.DoChan(key, fn) - async version returns channel
// - g.Forget(key) - allow new execution for key
type call struct {
	wg    sync.WaitGroup
	val   any
	err   error
	nDups int
}

// SingleFlight deduplicates function calls.
//
// TODO: Implement to:
// 1. Do(key, fn) executes fn only once per key at a time
// 2. Concurrent calls with same key wait for first to complete
// 3. All callers get the same result
// 4. After completion, new calls start fresh
type SingleFlight struct {
	mu    sync.Mutex
	calls map[string]*call
}

func NewSingleFlight() *SingleFlight {
	// YOUR CODE HERE
	return &SingleFlight{
		calls: map[string]*call{},
	}
}

// Do executes fn only if no other execution for key is in-flight.
// Returns (result, error, shared) where shared=true if result was shared.
func (sf *SingleFlight) Do(key string, fn func() (any, error)) (val any, err error, shared bool) {
	sf.mu.Lock()

	if c, inFlight := sf.calls[key]; inFlight {
		// if already in Flight
		c.nDups++
		sf.mu.Unlock()
		// Wait for it to complete
		c.wg.Wait()
		return c.val, c.err, true

	}
	// first initialise the wg for others to wait and indicate that we're going to call
	c := &call{}
	c.wg.Add(1)
	sf.calls[key] = c
	sf.mu.Unlock()

	// Cleanup
	defer func() {
		sf.mu.Lock()
		shared = c.nDups > 0
		c.wg.Done()
		delete(sf.calls, key)
		sf.mu.Unlock()
	}()

	// Do the call itself, all others can see its being done as key is set
	// What if this panics?
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panicked due to - %+v\n", r)
				c.err = fmt.Errorf("Panicked due to - %+v", r)
			}
		}()
		c.val, c.err = fn()
	}()
	return c.val, c.err, shared
}

// Forget removes key from the group, allowing a new call to start.
func (sf *SingleFlight) Forget(key string) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	delete(sf.calls, key)
}

// This use sync cond but is slower and not scalable as we lock the whole map to check if its done
type SingleFlightCond struct {
	mu    sync.Mutex
	calls map[string]*callCond
}

type callCond struct {
	done  bool
	cond  *sync.Cond
	val   any
	err   error
	nDups int
}

func NewSingleFlightCond() *SingleFlightCond {
	// YOUR CODE HERE
	return &SingleFlightCond{
		calls: map[string]*callCond{},
	}
}

func (sf *SingleFlightCond) DoSyncCond(key string, fn func() (any, error)) (val any, err error, shared bool) {
	sf.mu.Lock()

	if c, inFlight := sf.calls[key]; inFlight {
		// if already in Flight
		c.nDups++
		for !c.done {
			c.cond.Wait()
		}
		sf.mu.Unlock()

		return c.val, c.err, true

	}
	// first initialise the wg for others to wait and indicate that we're going to call
	c := &callCond{cond: sync.NewCond(&sf.mu)}
	sf.calls[key] = c
	sf.mu.Unlock()

	// Cleanup
	defer func() {
		sf.mu.Lock()
		shared = c.nDups > 0
		delete(sf.calls, key)
		c.done = true
		c.cond.Broadcast()
		sf.mu.Unlock()
	}()

	// Do the call itself, all others can see its being done as key is set
	// What if this panics?
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panicked due to - %+v\n", r)
				c.err = fmt.Errorf("Panicked due to - %+v", r)
			}
		}()
		c.val, c.err = fn()
	}()
	return c.val, c.err, shared
}

// =============================================================================
// PART 2: Async singleflight with DoChan
// =============================================================================

// SingleFlightResult holds the result of a DoChan call.
type SingleFlightResult struct {
	Val    any
	Err    error
	Shared bool
}

// DoChan is like Do but returns a channel that receives the result.
// This allows the caller to select on multiple operations or timeout.
//
// TODO: Implement to:
// 1. Return channel immediately
// 2. Execute fn (with deduplication)
// 3. Send result on channel when done
func (sf *SingleFlight) DoChan(key string, fn func() (any, error)) <-chan SingleFlightResult {
	// YOUR CODE HERE
	return nil
}

// =============================================================================
// PART 3: Practical singleflight Patterns
// =============================================================================

// CachedFetcher demonstrates the classic cache + singleflight pattern.
// This prevents cache stampede when cache expires.
//
// Cache stampede: Many requests arrive, find cache empty, all hit database.
// With singleflight: Only one request hits database, others wait and share.
type CachedFetcher struct {
	cache      map[string]cachedItem
	cacheMu    sync.RWMutex
	sf         *SingleFlight
	fetchCount int // For testing: how many actual fetches occurred
}

type cachedItem struct {
	value     string
	expiresAt time.Time
}

func NewCachedFetcher() *CachedFetcher {
	// YOUR CODE HERE
	return nil
}

// Fetch gets a value, using cache if available, otherwise fetching.
//
// TODO: Implement to:
// 1. Check cache first (with read lock)
// 2. If miss or expired, use singleflight to fetch
// 3. Update cache with result
// 4. Return value
//
// QUESTION: Where should you put the cache update - inside or outside singleflight?
func (cf *CachedFetcher) Fetch(key string, fetcher func(string) (string, error), ttl time.Duration) (result string, err error) {
	now := time.Now()
	cf.cacheMu.RLock()
	item, ok := cf.cache[key]
	cf.cacheMu.RUnlock()
	// if exists and not expired
	if ok && item.expiresAt.After(now) {
		return item.value, nil
	}

	// You only want to fetch + update the cache done in one atomic singleflight operation
	_, err, _ = cf.sf.Do(key, func() (any, error) {
		// Recheck if suddenly it hasn't been updated when computing expires time
		cf.cacheMu.RLock()
		item, ok = cf.cache[key]
		cf.cacheMu.RUnlock()
		if ok {
			result = item.value
			return nil, nil
		}

		val, e := fetcher(key)
		if e != nil {
			return nil, e
		}

		cf.cacheMu.Lock()
		cf.cache[key] = cachedItem{
			value:     val,
			expiresAt: time.Now().Add(ttl),
		}
		cf.fetchCount++
		cf.cacheMu.Unlock()

		result = val
		return nil, nil
	})
	return result, err
}

func (cf *CachedFetcher) FetchCount() int {
	return cf.fetchCount
}

// =============================================================================
// PART 4: singleflight with Timeout
// =============================================================================

// DoWithTimeout executes with deduplication and timeout.
//
// TODO: Implement to:
// 1. Use singleflight for deduplication
// 2. Return early if timeout exceeded
// 3. But don't cancel the underlying operation (let it complete for others)
//
// QUESTION: Should timeout cancel the operation or just stop waiting?
func (sf *SingleFlight) DoWithTimeout(key string, fn func() (any, error), timeout time.Duration) (any, error, bool) {
	// YOUR CODE HERE
	return nil, nil, false
}

// DoWithContext is like Do but respects context cancellation.
//
// TODO: Implement to:
// 1. Use singleflight for deduplication
// 2. Return early if context is cancelled
// 3. Let underlying operation complete for other waiters
func (sf *SingleFlight) DoWithContext(ctx context.Context, key string, fn func() (any, error)) (any, error, bool) {
	// YOUR CODE HERE
	return nil, nil, false
}

// =============================================================================
// PART 5: singleflight with Fresh Results
// =============================================================================

// Sometimes you want to ensure you get a fresh result, not a shared one.

// DoFresh forces a new execution even if one is in-flight.
// Existing waiters still get the in-flight result.
//
// TODO: Implement to:
// 1. Check if call is in-flight
// 2. If so, forget the key and start new
// 3. This caller gets fresh result, others get old one
func (sf *SingleFlight) DoFresh(key string, fn func() (any, error)) (any, error) {
	// YOUR CODE HERE
	return nil, nil
}

// =============================================================================
// PART 6: Request Coalescing
// =============================================================================

// RequestCoalescer batches requests within a time window.
// Similar to singleflight but collects multiple keys.
//
// Example: 100 requests for user IDs 1-100 arrive within 10ms.
// Instead of 100 DB queries, batch into one: SELECT * WHERE id IN (1..100)
type RequestCoalescer struct {
	// YOUR FIELDS HERE
}

type BatchResult struct {
	Results map[string]any
	Err     error
}

// NewRequestCoalescer creates a coalescer with given batch window.
//
// TODO: Implement to:
// 1. Collect requests within window duration
// 2. After window, call batchFn with all collected keys
// 3. Distribute results back to callers
func NewRequestCoalescer(window time.Duration, batchFn func(keys []string) (map[string]any, error)) *RequestCoalescer {
	// YOUR CODE HERE
	return nil
}

// Get requests a value, may be batched with other requests.
func (rc *RequestCoalescer) Get(key string) (any, error) {
	// YOUR CODE HERE
	return nil, nil
}

// =============================================================================
// CHALLENGE: Implement singleflight with stale-while-revalidate
// =============================================================================

// StaleWhileRevalidate serves stale cached data while fetching fresh.
// This is a common pattern in CDNs and caching layers.
//
// TODO: Implement to:
// 1. If cache hit and fresh, return immediately
// 2. If cache hit but stale, return stale AND trigger background refresh
// 3. If cache miss, fetch synchronously
// 4. Use singleflight for all fetches
//
// HINT: Need two TTLs - fresh duration and stale duration
type StaleWhileRevalidate struct {
	// YOUR FIELDS HERE
}

// func NewStaleWhileRevalidate(freshTTL, staleTTL time.Duration) *StaleWhileRevalidate {
// 	// YOUR CODE HERE
// 	return nil
// }

// func (swr *StaleWhileRevalidate) Get(key string, fetcher func(string) (string, error)) (string, error) {
// 	// YOUR CODE HERE
// 	return "", nil
// }

// Ensure imports are used
var (
	_ = context.Background
	_ = sync.Mutex{}
	_ = time.Second
)
