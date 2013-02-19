package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Quan és dissabte?")
	today := time.Now().Weekday()
	switch time.Saturday {
	case today + 0:
		fmt.Println("Avui.")
	case today + 1:
		fmt.Println("Demà.")
	case today + 2:
		fmt.Println("Demà passat.")
	default:
		fmt.Println("Massa tard.")
	}
}
