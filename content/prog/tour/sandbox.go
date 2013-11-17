package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	fmt.Println("Benvingut al playground!")

	fmt.Println("Ara són les", time.Now())

	fmt.Println("Si vols obrir un arxiu:")
	fmt.Println(os.Open("filename"))

	fmt.Println("O accedir a la xarxa:")
	fmt.Println(net.Dial("tcp", "google.com"))
}
