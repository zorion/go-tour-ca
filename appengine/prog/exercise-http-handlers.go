package main

import (
	"net/http"
)

func main() {
	// Les teves crides a http.Handle van aquí
	http.ListenAndServe("localhost:4000", nil)
}
