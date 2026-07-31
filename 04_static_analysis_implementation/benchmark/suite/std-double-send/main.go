package main
func sender(ch chan int) {
	ch <- 1
	ch <- 2
}
func receiver(ch chan int) {
	_ = <-ch
}
func main() {
	ch := make(chan int)
	go sender(ch)
	go receiver(ch)
	for {}
}
