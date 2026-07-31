package main

import "fmt"

func pinger(ping chan<- int, pong <-chan int) {
	for i := 0; i < 3; i++ {
		ping <- i
		<-pong
	}
}

func ponger(ping <-chan int, pong chan<- int) {
	for i := 0; i < 3; i++ {
		val := <-ping
		pong <- val
	}
}

func main() {
	ping := make(chan int)
	pong := make(chan int)
	go pinger(ping, pong)
	ponger(ping, pong)
	fmt.Println("Done")
}
