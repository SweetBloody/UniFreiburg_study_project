package main
func worker(ch chan int) { _ = <-ch }
func main() {
	ch := make(chan int)
	go worker(ch)
	for {}
}
