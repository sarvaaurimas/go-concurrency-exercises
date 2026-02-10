package main

import (
	"errors"
	"log"
	"sync"
	"time"
)

// Job represents a unit of work.
type Job struct {
	ID      int
	Payload int
}

// Result represents the output of processing a job.
type JobResult struct {
	JobID  int
	Output int
	Err    error
}

// =============================================================================
// PART 3: Dynamic Worker Pool
// =============================================================================

// DynamicPool adjusts worker count based on load.
//
// TODO: Implement a pool that:
// - Starts with minWorkers
// - Scales up to maxWorkers when jobs are queued
// - Scales down when idle for a period
//
// HINT: This is quite advanced! Consider:
// - How to detect load (queue depth)
// - How to spawn new workers safely
// - How to signal workers to exit
type DynamicPool struct {
	mu      sync.RWMutex
	jobs    chan Job
	results chan JobResult
	stopped bool
	wg      sync.WaitGroup
	once    sync.Once
	process func(Job) JobResult

	doneChans  []chan struct{}
	actW       int
	minW       int
	maxW       int
	maxCapPerc float64
}

func NewDynamicPool(minW, maxW, queuesize int, process func(Job) JobResult) *DynamicPool {
	wp := &DynamicPool{
		minW:       minW,
		maxW:       maxW,
		jobs:       make(chan Job, queuesize),
		results:    make(chan JobResult, queuesize),
		maxCapPerc: 0.8,
		process:    process,
	}
	for range minW {
		wp.NewWorker()
	}

	go func() {
		wp.Manage()
	}()

	return wp
}

func (wp *DynamicPool) NewWorker() {
	wp.actW++
	done := make(chan struct{}, 1)
	wp.doneChans = append(wp.doneChans, done)
	wp.wg.Go(func() {
		for {
			select {
			case job, ok := <-wp.jobs:
				if !ok {
					return
				}
				wp.results <- wp.process(job)

			// Shutdown if signalled
			case <-done:
				log.Println("Received shutdown signal")
				return
			}
		}
	})
}

func (wp *DynamicPool) StopWorker() {
	wp.actW--
	done := wp.doneChans[0]
	wp.doneChans = wp.doneChans[1:]
	done <- struct{}{}
}

// Manages workers and spins up and down dynamically
func (wp *DynamicPool) Manage() {
	for !wp.IsStopped() {
		// Spin up a new one if queue reached x%
		queueCurrCapPerc := wp.QueueLenCapPerc()
		log.Printf("Current cap - %.2f", queueCurrCapPerc)
		if queueCurrCapPerc > wp.maxCapPerc && wp.actW < wp.maxW {
			log.Printf("Cap reached %.2f, actW - %d, maxW - %d, spinning a new one", queueCurrCapPerc, wp.actW, wp.maxW)
			wp.NewWorker()
		}

		// Spin down one
		if queueCurrCapPerc == 0.0 && wp.actW > wp.minW {
			log.Printf("Queue 0, actW - %d, minW - %d,spinning down", wp.actW, wp.minW)
			wp.StopWorker()
		}

		time.Sleep(1 * time.Second)
	}
}

// Submit sends a job to the pool.
// Returns err if pool is shut or queue is full - drop msgs
func (wp *DynamicPool) Submit(job Job) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if wp.stopped {
		return errors.New("Worker pool shut down")
	}
	select {
	case wp.jobs <- job:
		return nil
	default:
		return errors.New("buffer queue full")
	}
}

// Results returns the channel to receive results from.
func (wp *DynamicPool) Results() <-chan JobResult {
	return wp.results
}

// Results returns the channel to receive results from.
func (wp *DynamicPool) IsStopped() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.stopped
}

// Returns what perc of total capacity is filled in the queue
func (wp *DynamicPool) QueueLenCapPerc() float64 {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return float64(len(wp.jobs)) / float64(cap(wp.jobs))
}

// Shutdown gracefully stops the pool.
// Waits for all in-flight jobs to complete.
//
// TODO: Implement to:
// 1. Close the jobs channel (no more submissions)
// 2. Wait for all workers to finish
// 3. Close the results channel
func (wp *DynamicPool) Shutdown() {
	wp.once.Do(func() {
		// Signal shutdown start
		wp.mu.Lock()
		wp.stopped = true
		// Close jobs to finish workers
		close(wp.jobs)
		wp.mu.Unlock()
		// Wait for workers to finish
		wp.wg.Wait()
		// Signal end
		close(wp.results)
	},
	)
}
