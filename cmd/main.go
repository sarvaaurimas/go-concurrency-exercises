package main

import (
	"fmt"
	"time"
)

func MergeChannels(ch1, ch2 <-chan int) <-chan int {
	merged := make(chan int)
	closed := make(chan struct{})
	done := make(chan struct{})
	go func() {
		for {
			select {
			case val, open := <-ch1:
				if !open {
					ch1 = nil
					closed <- struct{}{}
					fmt.Println("Ch1 closed")
					continue
				}
				merged <- val
			case val, open := <-ch2:
				if !open {
					ch2 = nil
					closed <- struct{}{}
					fmt.Println("Ch2 closed")
					continue
				}
				merged <- val
			case <-done:
				fmt.Println("Cleaning up")
				return
			}
		}
	}()

	go func() {
		for range 2 {
			<-closed
		}
		done <- struct{}{}
		close(merged)
	}()

	return merged
}

func main() {
	ch1, ch2 := make(chan int), make(chan int)

	go func() {
		defer close(ch1)
		for i := range 5 {
			ch1 <- i
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		defer close(ch2)
		for i := range 5 {
			ch1 <- i
			time.Sleep(1 * time.Second)
		}
	}()

	merged := MergeChannels(ch1, ch2)

	for msg := range merged {
		fmt.Println(msg)
	}
	time.Sleep(1 * time.Second)
}
