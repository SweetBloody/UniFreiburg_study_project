package main

import "fmt"

func worker(in <-chan int, done chan<- bool) {
	for range in {
		// processing
	}
	done <- true
}

func main() {
	jobs := make(chan int)
	done := make(chan bool)

	go worker(jobs, done)
	go worker(jobs, done)

	jobs <- 100
	jobs <- 200
	close(jobs)

	<-done
	<-done
	fmt.Println("All workers finished")
}
