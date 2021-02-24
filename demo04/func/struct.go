package main

import "fmt"

type MyStruct struct {
	i int
}

func mf1(a MyStruct, b *MyStruct) {
	a.i = 100
	b.i = 200
	fmt.Printf("in my_function - a=(%d, %p) b=(%v, %p)\n", a, &a, b, &b)
}

func main() {
	a := MyStruct{i: 10}
	b := &MyStruct{i: 20}
	fmt.Printf("before calling - a=(%d, %p) b=(%v, %p)\n", a, &a, b, &b)
	mf1(a, b)
	fmt.Printf("after calling  - a=(%d, %p) b=(%v, %p)\n", a, &a, b, &b)
}
