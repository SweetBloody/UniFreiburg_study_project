package main
func sender(ch chan int) {
	ch <- 10
	ch <- 20
}
func receiver(ch chan int) {
	for v := range ch {
		_ = v
	}
}
func main() {
	ch := make(chan int)
	go sender(ch)
	go receiver(ch)
	for {}
}
