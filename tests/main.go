package main

import (
	"fmt"
	"gowasm/tests/fac"
	"gowasm/tests/i32"
)

func main() {
	fmt.Printf("Starting tests...\n")
	v64 := fac.Fact(0)
	fmt.Printf("-- Asserting return... Fact(0) --> %d\n", v64)
	v64 = fac.Fact(3)
	fmt.Printf("-- Asserting return... Fact(3) --> %d\n", v64)
	fac.PrintAll(3)
	v32 := i32.Add(1, 5)
	fmt.Printf("-- Asserting return... Add(1, 5) --> %d\n", v32)
	v32 = i32.Expr1(10, 3)
	fmt.Printf("-- Asserting return... Expr1(10, 3) --> %d\n", v32)
	v32 = i32.Expr2(100, 20, 5)
	fmt.Printf("-- Asserting return... Expr2(100, 20, 5) --> %d\n", v32)
	fmt.Printf("Tests complete\n")
}
