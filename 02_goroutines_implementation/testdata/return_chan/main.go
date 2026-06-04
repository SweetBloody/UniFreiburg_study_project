package main

func reader(ch chan int) {
	<-ch
}

func create() chan int {
	a := make(chan int)
	return a
}

func writer(ch chan int) {
	ch <- 1
	close(ch)
}

func main() {
	c := make(chan int)

	go reader(c)
	go writer(c)

	b := create()
	go reader(b)
	go writer(b)
}
