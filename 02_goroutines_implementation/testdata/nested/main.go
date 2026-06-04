package main

func process(ch chan int) {
	doWork(ch)

	go generator(ch)
}

func doWork(ch chan int) {
	for v := range ch {
		_ = v
	}
}

func generator(ch chan int) {
	for i := 0; i < 5; i++ {
		ch <- i
	}
	close(ch)
}

func main() {
	c := make(chan int)

	go generator(c)

	go process(c)
}
