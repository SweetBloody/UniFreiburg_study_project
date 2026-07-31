package main
func p1(c1, c2 chan int) {
	c1 <- 1
	_ = <-c2
}
func p2(c1, c2 chan int) {
	c2 <- 1
	_ = <-c1
}
func main() {
	c1 := make(chan int)
	c2 := make(chan int)
	go p1(c1, c2)
	go p2(c1, c2)
	for {}
}
