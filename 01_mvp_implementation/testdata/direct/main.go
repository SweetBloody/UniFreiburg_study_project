package main

func worker(ch chan int) {}

func main() {
	c := make(chan int)
	worker(c)
}
