package main

func source1(out chan int) {
	out <- 1
}

func source2(out chan int) {
	out <- 2
}

func consumer(in chan int) {
	<-in
	<-in
}

func main() {
	c := make(chan int)

	go source1(c)
	go source2(c)
	go consumer(c)
}
