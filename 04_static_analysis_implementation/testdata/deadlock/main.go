package main

import "fmt"

func sender(ch chan int) {
	fmt.Println("Sender: trying to write 10")
	ch <- 10
	fmt.Println("Sender: trying to write 20 (will deadlock here!)")
	ch <- 20
}

func receiver(ch chan int) {
	fmt.Println("Receiver: reading first value")
	val := <-ch
	fmt.Println("Receiver got:", val)
}

func main() {
	ch := make(chan int)

	go sender(ch)
	go receiver(ch)

	for {
	}
}