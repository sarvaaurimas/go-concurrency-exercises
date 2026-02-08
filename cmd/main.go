package main

import (
	"fmt"
	"sync"
)

func FetchAll(urls []string, fetcher func(string) string) map[string]string {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]string)

	for _, url := range urls {
		wg.Go(
			func() {
				result := fetcher(url)
				mu.Lock()
				results[url] = result
				mu.Unlock()
			},
		)
	}
	wg.Wait()
	return results
}

func Fetch(url string) string {
	return url
}

func main() {
	urls := []string{"hi", "hello", "yo"}
	results := FetchAll(urls, Fetch)
	fmt.Println(results)
}
