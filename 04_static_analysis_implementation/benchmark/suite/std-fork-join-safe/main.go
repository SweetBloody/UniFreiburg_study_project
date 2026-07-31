package main

import "fmt"

func worker(id int, ch chan<- int) {
	ch <- id * 10
}

func main() {
	ch := make(chan int)
	go worker(1, ch)
	go worker(2, ch)

	res1 := <-ch
	res2 := <-ch
	fmt.Println(res1 + res2)
}
