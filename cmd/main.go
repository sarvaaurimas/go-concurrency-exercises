package main

import "fmt"

func main() {
	a := make([]int, 4)
	fmt.Println(a)
	a = a[:0]
	fmt.Println(a)
}
