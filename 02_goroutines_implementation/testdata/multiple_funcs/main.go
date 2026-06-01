package main

func worker1(ch chan int) {}

func worker2(ch chan int) {}

func main() {
	a := make(chan int)
	b := make(chan int)

	worker1(a)
	worker2(a)
	worker2(b)
}
