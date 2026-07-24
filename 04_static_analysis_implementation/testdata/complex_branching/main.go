package main

func worker(id int, in chan int, out chan string) {
	for v := range in {
		if v%2 == 0 {
			for i := 0; i < 2; i++ {
				out <- "even"
			}
		} else {
			out <- "odd"
		}
	}
}

func main() {
	in := make(chan int)
	out := make(chan string)

	for w := 0; w < 3; w++ {
		go worker(w, in, out)
	}

	for i := 0; i < 10; i++ {
		in <- i
	}
	close(in)

	for j := 0; j < 10; j++ {
		<-out
	}
}
