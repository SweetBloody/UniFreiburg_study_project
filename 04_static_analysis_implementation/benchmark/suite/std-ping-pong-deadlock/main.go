package main

import "fmt"

func pinger(ping <-chan int, pong chan<- int) {
	val := <-ping
	pong <- val
}

func ponger(pong <-chan int, ping chan<- int) {
	val := <-pong
	ping <- val
}

func main() {
	ping := make(chan int)
	pong := make(chan int)
	go pinger(ping, pong)
	ponger(pong, ping)
	fmt.Println("Done")
}
