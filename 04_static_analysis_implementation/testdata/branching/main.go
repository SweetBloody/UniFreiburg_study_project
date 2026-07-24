package main

import "os"

func reader(c chan int) {
	<-c
}

func main() {
	c1 := make(chan int)
	c2 := make(chan int)

	var target chan int
	if len(os.Args) > 1 {
		target = c1
	} else {
		target = c2
	}

	go reader(target)

	target <- 1
}
