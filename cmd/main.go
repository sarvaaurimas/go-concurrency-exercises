package main

import (
	"log"
	"sync"
	"time"
)

func FakeDBReturn(sql string) (string, error) {
	time.Sleep(500 * time.Millisecond)
	return time.Now().String(), nil
	// return sql, nil
}

func main() {
	var wg sync.WaitGroup
	flight := NewSingleFlight()
	query := "Hello Query"
	for i := range 10 {
		if i == 5 {
			time.Sleep(300 * time.Millisecond)
			log.Println("Forgetting")
			flight.Forget("Hello")
		}
		wg.Go(func() {
			res, err, dup := flight.Do("Hello", func() (any, error) {
				return FakeDBReturn(query)
			})
			log.Printf("Result - %s, err - %v, duplicates - %v\n", res, err, dup)
		})
	}
	wg.Wait()
}
