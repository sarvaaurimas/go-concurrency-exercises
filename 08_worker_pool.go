package concurrency

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// =============================================================================
// EXERCISE 3.1: Worker Pool Pattern
// =============================================================================
//
// A worker pool manages a fixed number of goroutines that process jobs.
// This is essential for controlling resource usage and preventing goroutine explosion.
//
// KEY CONCEPTS:
// - Fixed number of workers prevents unbounded goroutine creation
// - Jobs are sent to a channel, workers receive and process them
// - Results can be collected via a separate channel
// - Graceful shutdown requires coordinating job submission and worker termination
//
// =============================================================================

// =============================================================================
// PART 1: Basic Worker Pool
// =============================================================================

// NewWorkerPool creates a pool with n workers.
// The process function is called for each job.
//
// TODO: Implement to:
// 1. Create job and result channels
// 2. Start n worker goroutines
// 3. Each worker loops: receive job, process, send result
// 4. Workers should exit when jobs channel is closed

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

// BufferedPool has an internal job queue for backpressure handling.
//
// TODO: Implement a pool where:
// - Submit never blocks (adds to internal queue)
// - Workers pull from the queue
// - Queue has maximum size (reject jobs when full)
//
// QUESTION: How would you handle the case when the queue is full?

// TODO: Implement a worker pool that:
// - Has a configurable number of workers
// - Accepts jobs via a channel
// - Returns results via a channel
// - Supports graceful shutdown
type BufferedPool struct {
	mu      sync.Mutex
	jobs    chan Job
	results chan JobResult
	stopped bool
	wg      *sync.WaitGroup
	once    sync.Once
}

func NewWorkerPool(n int, queuesize int, process func(Job) JobResult) *BufferedPool {
	wp := &BufferedPool{
		jobs:    make(chan Job, queuesize),
		results: make(chan JobResult, queuesize),
	}
	for range n {
		// Worker function
		wp.wg.Go(func() {
			for job := range wp.jobs {
				wp.results <- process(job)
			}
		})
	}
	return wp
}

// Submit sends a job to the pool.
// Returns err if pool is shut or queue is full - drop msgs
func (wp *BufferedPool) Submit(job Job) error {
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
func (wp *BufferedPool) Results() <-chan JobResult {
	return wp.results
}

// Shutdown gracefully stops the pool.
// Waits for all in-flight jobs to complete.
//
// TODO: Implement to:
// 1. Close the jobs channel (no more submissions)
// 2. Wait for all workers to finish
// 3. Close the results channel
func (wp *BufferedPool) Shutdown() {
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

// =============================================================================
// CHALLENGE: Implement a pool with priorities
// =============================================================================

// PriorityJob includes a priority level.
type PriorityJob struct {
	Job
	Priority int // Higher = more urgent
}

// PriorityPool processes higher priority jobs first.
//
// TODO: Implement a pool that:
// - Maintains separate queues per priority level
// - Workers always take from highest priority non-empty queue
// - Lower priority jobs don't starve (optional: implement aging)
//
// HINT: You might need a heap or multiple channels
type PriorityPool struct {
	// YOUR FIELDS HERE
}
