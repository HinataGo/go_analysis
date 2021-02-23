package string

import (
	"fmt"
	"testing"
)

func TestString(t *testing.T) {
	str := "string"
	println([]byte(str))
	fmt.Printf("%T", str[0])

}
