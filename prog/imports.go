package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Printf("Ara tens %g problemes.",
		math.Nextafter(2, 3))
}
