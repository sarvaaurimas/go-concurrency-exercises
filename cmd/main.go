package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	minW := 2
	maxW := 10
	queuesize := 50
	// Spin up a new one when it reaches 80%
	pool := NewDynamicPool(minW, maxW, queuesize, func(Job) JobResult {
		time.Sleep(500 * time.Millisecond)
		return JobResult{}
	},
	)

	go func() {
		for range queuesize * 10 {
			time.Sleep(100 * time.Millisecond)
			log.Println(len(pool.jobs))
			err := pool.Submit(Job{})
			if err != nil {
				log.Println(err)
			}
		}
		fmt.Println("Done submitting")
	}()

	go func() {
		time.Sleep(60 * time.Second)
		fmt.Println("Shutting Down")
		pool.Shutdown()
	}()

	// Consumer of results
	wg.Go(func() {
		for range pool.Results() {
			log.Println("Received Job")
		}
	})
	wg.Wait()
}
