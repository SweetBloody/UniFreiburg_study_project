package main
func sender(ch chan int) {
	ch <- 1
}
func receiver(ch chan int) {
	_ = <-ch
	_ = <-ch
}
func main() {
	ch := make(chan int)
	go sender(ch)
	go receiver(ch)
	for {}
}
