package main

import (
	"fmt"
	"gowasm/tests/fac"
)

func main() {
	fmt.Printf("Starting tests...\n")
	i64 := fac.Add(13, 200)
	fmt.Printf("-- Asserting return... Add(13, 200) --> %d\n", i64)
	fmt.Printf("Tests complete\n")
}
