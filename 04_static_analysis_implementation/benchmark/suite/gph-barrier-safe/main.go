package main

import "fmt"

func worker(id int, arrive chan<- int, depart <-chan int) {
	arrive <- id
	<-depart
}

func coordinator(arrive <-chan int, depart chan<- int) {
	<-arrive
	<-arrive
	depart <- 1
	depart <- 1
}

func main() {
	arrive := make(chan int)
	depart := make(chan int)

	go worker(10, arrive, depart)
	go worker(20, arrive, depart)
	coordinator(arrive, depart)
	fmt.Println("Barrier passed")
}
