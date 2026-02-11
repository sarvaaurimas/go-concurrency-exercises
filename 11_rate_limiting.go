package concurrency

import (
	"context"
	"sync"
	"time"
)

// =============================================================================
// EXERCISE 3.4: Rate Limiting
// =============================================================================
//
// Rate limiting controls how frequently operations can occur.
// Essential for API calls, resource protection, and fair usage.
//
// KEY CONCEPTS:
// - time.Ticker provides steady rate limiting
// - Token bucket allows bursts up to a limit
// - Leaky bucket smooths out bursts
// - golang.org/x/time/rate provides production-ready implementation
//
// =============================================================================

// =============================================================================
// PART 1: Simple Rate Limiter with Ticker
// =============================================================================

// TickerLimiter allows one operation per interval.
//
// TODO: Implement using time.Ticker
// - Wait() blocks until next tick
// - Stop() cleans up the ticker
//
// QUESTION: What happens if you call Wait() faster than the interval?
type TickerLimiter struct {
	ticker *time.Ticker
	done   chan struct{}
}

// NewTickerLimiter creates a limiter allowing one op per interval.
func NewTickerLimiter(interval time.Duration) *TickerLimiter {
	// YOUR CODE HERE
	return &TickerLimiter{
		ticker: time.NewTicker(interval),
	}
}

// Wait blocks until the next operation is allowed.
func (tl *TickerLimiter) Wait() {
	select {
	case <-tl.ticker.C:
	case <-tl.done:
	}
}

// Stop releases resources.
func (tl *TickerLimiter) Stop() {
	tl.ticker.Stop()
	close(tl.done)
}

// =============================================================================
// PART 2: Token Bucket Rate Limiter
// =============================================================================

// TokenBucket implements token bucket algorithm.
// Allows bursts up to bucket capacity, refills at steady rate.
//
// TODO: Implement token bucket:
// - Bucket holds up to 'capacity' tokens
// - Tokens are added at 'rate' tokens per second
// - Take(n) removes n tokens, blocking if insufficient
// - TryTake(n) is non-blocking, returns false if insufficient
//
// QUESTION: How does token bucket differ from ticker limiter?
type TokenBucket struct {
	tokens chan struct{}
}

// NewTokenBucket creates a bucket with given capacity and refill rate.
// Starts full.
func NewTokenBucket(ctx context.Context, capacity int, refillRate float64) *TokenBucket {
	tb := TokenBucket{
		tokens: make(chan struct{}, capacity),
	}
	tickInt := time.Duration(float64(time.Second) / refillRate)
	ticker := time.Tick(tickInt)

	// Fill the bucket first
	for range capacity {
		tb.tokens <- struct{}{}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker:
				select {
				case tb.tokens <- struct{}{}:
				// Drop if full
				default:
				}
			}
		}
	}()
	return &tb
}

// Take removes n tokens, blocking until available.
func (tb *TokenBucket) Take(n int) {
	for range n {
		<-tb.tokens
	}
}

// TryTake attempts to remove n tokens without blocking.
// Returns true if successful.
func (tb *TokenBucket) TryTake(n int) bool {
	var taken int
	for range n {
		select {
		case <-tb.tokens:
			taken++
		default:
			// Return what we've taken
			for range taken {
				select {
				case tb.tokens <- struct{}{}:
				default:
					// Drop if it has already been refilled to max capacity
				}
			}
			return false
		}
	}
	return true
}

// Available returns current number of tokens (approximate).
func (tb *TokenBucket) Available() int {
	return len(tb.tokens)
}

// =============================================================================
// PART 3: Rate Limiter with Context
// =============================================================================

// ContextLimiter respects cancellation while rate limiting.
//
// TODO: Implement to:
// - WaitN(ctx, n) waits for n tokens, respects cancellation
// - Returns error if context cancelled before tokens available
type ContextLimiter struct {
	ticker <-chan time.Time
}

// NewContextLimiter creates a limiter with given rate (ops per second).
func NewContextLimiter(rate float64) *ContextLimiter {
	if rate <= 0 {
		panic("Rate leq 0")
	}
	ticker := time.Tick(time.Duration(float64(time.Second) / rate))
	return &ContextLimiter{ticker: ticker}
}

// WaitN waits for n tokens, respecting context cancellation.
func (cl *ContextLimiter) WaitN(ctx context.Context, n int) error {
	for range n {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cl.ticker:
		}
	}
	return nil
}

// =============================================================================
// PART 4: Per-Key Rate Limiter
// =============================================================================

// PerKeyLimiter maintains separate rate limits per key.
// Useful for per-user or per-IP rate limiting.
//
// TODO: Implement to:
// - Each key has its own token bucket
// - Keys are created on first access
// - Stale keys can be cleaned up (optional)
//
// QUESTION: What's the memory concern with per-key limiters?

type UserLimiter struct {
	userBucket *TokenBucket
	expDate    time.Time
}

type PerKeyLimiter struct {
	mu            sync.Mutex
	usersLimiters map[string]*UserLimiter
	capacity      int
	refillRate    float64
	expDuration   time.Duration
}

// NewPerKeyLimiter creates a per-key limiter.
// Each key gets a bucket with given capacity and rate.
func NewPerKeyLimiter(capacity int, refillRate float64, expDuration time.Duration) *PerKeyLimiter {
	return &PerKeyLimiter{
		capacity:      capacity,
		refillRate:    refillRate,
		expDuration:   expDuration,
		usersLimiters: map[string]*UserLimiter{},
	}
}

// Allow checks if operation is allowed for given key.
// Consumes one token if allowed.
func (pkl *PerKeyLimiter) Allow(key string) bool {
	pkl.mu.Lock()
	userLimiter, ok := pkl.usersLimiters[key]
	if !ok {
		// Initialise new one
		pkl.usersLimiters[key] = &UserLimiter{
			userBucket: NewTokenBucket(context.Background(), pkl.capacity, pkl.refillRate),
		}
	}
	userLimiter.expDate = time.Now().Add(pkl.expDuration)
	pkl.mu.Unlock()
	return userLimiter.userBucket.TryTake(1)
}

// =============================================================================
// PART 5: Leaky Bucket (Smoothing)
// =============================================================================

// LeakyBucket processes items at a steady rate, dropping excess.
// Unlike token bucket, it smooths out bursts completely.
//
// TODO: Implement to:
// - Items are added to a queue
// - Queue is drained at steady rate
// - If queue is full, new items are dropped
// - Process function is called for each item at steady rate
type LeakyBucket struct {
	queue  chan int
	ticker <-chan time.Time

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
}

// NewLeakyBucket creates a bucket with given queue size and drain rate.
func NewLeakyBucket(queueSize int, drainRate float64) *LeakyBucket {
	if drainRate <= 0.0 {
		panic("Invalid drain rate")
	}

	if queueSize <= 0 {
		panic("Invalid queue size")
	}
	return &LeakyBucket{
		queue:  make(chan int, queueSize),
		ticker: time.Tick(time.Duration(float64(time.Second) / drainRate)),
	}
}

// Add adds an item to the bucket.
// Returns false if bucket is full (item dropped).
func (lb *LeakyBucket) Add(item int) bool {
	select {
	case lb.queue <- item:
		return true
	default:
		return false
	}
}

// Start begins draining the bucket, calling process for each item.
func (lb *LeakyBucket) Start(process func(int)) {
	// Initialise for in case started is called
	lb.mu.Lock()
	if lb.started {
		lb.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	lb.cancel = cancel
	lb.started = true
	lb.mu.Unlock()

	go func() {
		for {
			select {
			case <-lb.ticker:
				select {
				case item := <-lb.queue:
					process(item)
				default:
					// Only pick one if available
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the bucket processing.
func (lb *LeakyBucket) Stop() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if lb.started {
		lb.cancel()
		lb.started = false
	}
}

// =============================================================================
// CHALLENGE: Implement adaptive rate limiter
// =============================================================================

// AdaptiveLimiter adjusts rate based on feedback.
//
// TODO: Implement to:
// - Increase rate when Success() is called
// - Decrease rate when Failure() is called
// - Stay within min/max bounds
// - Use exponential backoff on failure
//
// This is useful for handling varying server capacity.
type AdaptiveLimiter struct {
	// YOUR FIELDS HERE
}

// NewAdaptiveLimiter creates a limiter that adjusts between min and max rate.
// func NewAdaptiveLimiter(minRate, maxRate, initialRate float64) *AdaptiveLimiter {
// 	// YOUR CODE HERE
// 	return nil
// }

// Ensure imports are used
var (
	_ = context.Background
	_ = time.Second
)
