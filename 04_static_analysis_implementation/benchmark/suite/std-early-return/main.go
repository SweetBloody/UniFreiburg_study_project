package main
func worker(ch chan int, ok bool) {
	if !ok {
		return
	}
	ch <- 1
}
func main() {
	ch := make(chan int)
	go worker(ch, false)
	_ = <-ch
	for {}
}
