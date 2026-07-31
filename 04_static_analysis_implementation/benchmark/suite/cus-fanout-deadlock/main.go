package main

import "fmt"

func workerLimited(in <-chan int, done chan<- bool) {
	<-in
	done <- true
}

func main() {
	jobs := make(chan int)
	done := make(chan bool)

	go workerLimited(jobs, done)
	go workerLimited(jobs, done)

	jobs <- 100
	jobs <- 200
	jobs <- 300
	close(jobs)

	<-done
	<-done
	fmt.Println("All workers finished")
}
