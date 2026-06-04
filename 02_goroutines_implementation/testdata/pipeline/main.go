package main

func generator(out chan int) {
	for i := 0; i < 5; i++ {
		out <- i
	}
	close(out)
}

func squarer(in chan int, out chan int) {
	for v := range in {
		out <- v * v
	}
	close(out)
}

func printer(in chan int) {
	for v := range in {
		_ = v
	}
}

func main() {
	c1 := make(chan int)
	c2 := make(chan int)

	go generator(c1)
	go squarer(c1, c2)
	go printer(c2)
}
