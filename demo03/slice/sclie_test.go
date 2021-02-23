package slice

import (
	"fmt"
	"testing"
)

func TestSlice(t *testing.T) {
	a := make([]int, 2)
	b := []int{1, 2}
	copy(a, b)
	fmt.Printf("%v", a)
}
