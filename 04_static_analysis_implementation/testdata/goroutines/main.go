package main

func reader(ch chan int) {
	<-ch
}

func writer(ch chan int) {
	ch <- 1
	close(ch)
}

func main() {
	c := make(chan int)

	go reader(c)
	go writer(c)
}
