package main

import (
	"fmt"
	"gowasm/tests/fac"
	"gowasm/tests/i32"
)

func main() {
	fmt.Printf("Starting tests...\n")
	i64 := fac.Add(13, 200)
	fmt.Printf("-- Asserting return... Add(13, 200) --> %d\n", i64)
	i32 := i32.Add(1, 5)
	fmt.Printf("-- Asserting return... Add(1, 5) --> %d\n", i32)
	fmt.Printf("Tests complete\n")
}
