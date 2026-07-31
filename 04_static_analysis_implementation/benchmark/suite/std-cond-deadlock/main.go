package main
func sender(ch chan int, flag bool) {
	if flag {
		ch <- 100
	}
}
func receiver(ch chan int) {
	_ = <-ch
}
func main() {
	ch := make(chan int)
	go sender(ch, false)
	go receiver(ch)
	for {}
}
