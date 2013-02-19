package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	p = Vertex{1, 2}  // tipus Vertex
	q = &Vertex{1, 2} // tipus *Vertex (punter)
	r = Vertex{X: 1}  // Y:0 és implicit
	s = Vertex{}      // X:0 i Y:0
)

func main() {
	fmt.Println(p, q, r, s)
}
