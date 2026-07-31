package main
func gen(out chan int) {
	out <- 1
	out <- 2
	close(out)
}
func filter(in chan int, out chan int) {
	for v := range in {
		out <- v * 2
	}
}
func read(in chan int) {
	for v := range in {
		_ = v
	}
}
func main() {
	c1 := make(chan int)
	c2 := make(chan int)
	go gen(c1)
	go filter(c1, c2)
	go read(c2)
	for {}
}
