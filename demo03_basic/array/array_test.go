package array

import (
	"fmt"
	"testing"
	"unsafe"
)

func array() {

}

type e struct {
	a int
	b string
}

func TestArr(t *testing.T) {
	e1 := e{
		a: 1,
		b: "11",
	}
	e2 := e{
		a: 2,
		b: "112",
	}
	var arr [2]e
	arr[0] = e1
	arr[1] = e2
	//
	fmt.Printf("arr address:%p \n", &arr)
	fmt.Printf("arr[0] address:%p \n", &arr[0])
	fmt.Println("arr[0] size: ", unsafe.Sizeof(arr[0]))
	fmt.Printf("arr[1] address:%p  \n", &arr[1])
	fmt.Println("arr[1] size: ", unsafe.Sizeof(arr[1]))

	fmt.Println("-----------------------------------------")
	a := [2]int{5, 6}
	b := [2]int{5, 6}
	// 这里即使ab地址不同，但是,   go有这么一句话
	// Array values are comparable if values of the array element type are comparable. Two array values are equal if their corresponding elements are equal.
	// array 中元素一样那么，他们比较就是相等的，这里不是对比地址
	fmt.Printf("a : %p \n", &a)
	fmt.Printf("b : %p \n", &b)
}
