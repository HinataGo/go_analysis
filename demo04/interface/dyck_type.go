package main

import "fmt"

type Duck interface {
	Color()
	Type()
}

type Cat struct {
	Name string
}

//go:noinline
func (c Cat) Color() {
	fmt.Println("color")
}

func (c Cat) Type() {
	fmt.Println("type")
}

func main() {
	// c := &Cat{}
	// c.Color()
	// c.Type()
	// var d Duck = &Cat{}
	// d.Color()

	// interface --> 具体类型
	var c1 Duck = &Cat{Name: "little cat "}
	switch c1.(type) {
	case *Cat:
		cat := c1.(*Cat)
		cat.Color()
	}
}
