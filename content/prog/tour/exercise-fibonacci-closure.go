package main

import "fmt"

// fibonacci és una funció que torna
// una funció que torna un int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
