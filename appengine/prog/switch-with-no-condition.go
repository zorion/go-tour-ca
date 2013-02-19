package main

import (
	"fmt"
	"time"
)

func main() {
	t := time.Now()
	switch {
	case t.Hour() < 12:
		fmt.Println("Bon dia!")
	case t.Hour() < 19:
		fmt.Println("Bona tarda.")
	default:
		fmt.Println("Bon vespre.")
	}
}
