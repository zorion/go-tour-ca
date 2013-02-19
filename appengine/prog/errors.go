package main

import (
	"fmt"
	"time"
)

type MyError struct {
	When time.Time
	What string
}

func (e *MyError) Error() string {
	return fmt.Sprintf("A les %v, %s",
		e.When, e.What)
}

func run() error {
	return &MyError{
		time.Now(),
		"ha fallat",
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}
