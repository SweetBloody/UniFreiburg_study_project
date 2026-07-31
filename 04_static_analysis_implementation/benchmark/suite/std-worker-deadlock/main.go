package main
func worker1(ch chan int) { _ = <-ch }
func worker2(ch chan int) { _ = <-ch }
func main() {
	ch := make(chan int)
	go worker1(ch)
	go worker2(ch)
	for {}
}
