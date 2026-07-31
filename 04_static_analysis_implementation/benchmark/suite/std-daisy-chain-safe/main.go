package main

import "fmt"

func stage(in <-chan int, out chan<- int) {
	val := <-in
	out <- val + 1
}

func sender(out chan<- int) {
	out <- 10
}

func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)

	go sender(ch1)
	go stage(ch1, ch2)
	go stage(ch2, ch3)

	res := <-ch3
	fmt.Println(res)
}
