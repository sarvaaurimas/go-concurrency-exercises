package main

import (
	"fmt"
	"log"
	"sync"
)

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
	log.Println("Starting")
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
