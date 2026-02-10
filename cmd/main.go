package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	b := NewBarrier(20)

	for i := range 20 {
		time.Sleep(1000 * time.Millisecond)
		wg.Go(func() {
			fmt.Printf("Goroutine %d started\n", i)
			b.Wait()
			fmt.Printf("Goroutine %d resumed\n", i)
			time.Sleep(1000 * time.Millisecond)
			fmt.Printf("Goroutine %d started\n", i)
			b.Wait()
			fmt.Printf("Goroutine %d resumed\n", i)
		},
		)
	}
	wg.Wait()
}
