package main

func createMulti() (chan int, chan string) {
	c1 := make(chan int)
	c2 := make(chan string)
	return c1, c2
}

func reader(c chan int) {
	<-c
}

func writerStr(c chan string) {
	c <- "hello"
}

func main() {
	chInt, chStr := createMulti()

	go reader(chInt)
	go writerStr(chStr)
}
