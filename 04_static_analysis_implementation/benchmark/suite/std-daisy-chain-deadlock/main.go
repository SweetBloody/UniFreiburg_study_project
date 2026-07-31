package main

import "fmt"

func stageBroken(in <-chan int, out chan<- int) {
	val1 := <-in
	val2 := <-in
	out <- val1 + val2
}

func sender(out chan<- int) {
	out <- 10
}

func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go sender(ch1)
	go stageBroken(ch1, ch2)

	res := <-ch2
	fmt.Println(res)
}
