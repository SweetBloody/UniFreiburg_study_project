package main

func worker(ch chan int) {}

func main() {
	a := make(chan int)
	b := make(chan int)

	worker(a)
	worker(b)
}
