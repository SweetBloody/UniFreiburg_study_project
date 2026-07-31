package main
func sender(ch chan int) {
	for i := 0; i < 5; i++ {
		ch <- i
	}
}
func receiver(ch chan int) {
	for i := 0; i < 3; i++ {
		_ = <-ch
	}
}
func main() {
	ch := make(chan int)
	go sender(ch)
	go receiver(ch)
	for {}
}
