package main

import "fmt"

func worker1(ch chan<- int) {
	ch <- 10
}

func worker2(ch chan<- int, skip bool) {
	if skip {
		return
	}
	ch <- 20
}

func main() {
	ch := make(chan int)
	go worker1(ch)
	go worker2(ch, true)

	res1 := <-ch
	res2 := <-ch
	fmt.Println(res1 + res2)
}
