package main

import "fmt"

func main() {
	m := make(map[string]int)

	m["Resposta"] = 42
	fmt.Println("Valor:", m["Resposta"])

	m["Resposta"] = 48
	fmt.Println("Valor:", m["Resposta"])

	delete(m, "Resposta")
	fmt.Println("Valor:", m["Resposta"])

	v, ok := m["Resposta"]
	fmt.Println("Valor:", v, "Existeix?", ok)
}
