package main

func worker(ch chan int) {
	<-ch
}

func main() {
	c := make(chan int)
	x := c
	worker(x)
}
