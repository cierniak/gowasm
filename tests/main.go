package main

import (
	"fmt"
	"gowasm/rt/gc"
	"gowasm/tests/fac"
	"gowasm/tests/i32"
	"gowasm/tests/mem"
	"gowasm/tests/newstuff"
)

func main() {
	fmt.Printf("Starting tests...\n")
	v64 := fac.Fact(0)
	fmt.Printf("-- Asserting return... fac.Fact(0) --> %d\n", v64)
	v64 = fac.Fact(3)
	fmt.Printf("-- Asserting return... fac.Fact(3) --> %d\n", v64)
	fac.PrintAll(3)
	v32 := i32.Add(1, 5)
	fmt.Printf("-- Asserting return... i32.Add(1, 5) --> %d\n", v32)
	v32 = i32.Expr1(10, 3)
	fmt.Printf("-- Asserting return... i32.Expr1(10, 3) --> %d\n", v32)
	v32 = i32.Expr2(100, 20, 5)
	fmt.Printf("-- Asserting return... i32.Expr2(100, 20, 5) --> %d\n", v32)
	v32 = i32.DivSigned(100, 20)
	fmt.Printf("-- Asserting return... i32.DivSigned(100, 20) --> %d\n", v32)
	v32 = i32.NestedLoop(5, 7)
	fmt.Printf("-- Asserting return... i32.NestedLoop(5, 7) --> %d\n", v32)
	uip := i32.AddUintPtr(5, 3)
	fmt.Printf("-- Asserting return... i32.AddUintPtr(5, 3) --> %d\n", uip)
	v32 = mem.R(16, 8)
	fmt.Printf("-- Asserting return... mem.R(16, 8) --> %d\n", v32)
	v32 = mem.F(6)
	fmt.Printf("-- Asserting return... mem.F(6) --> %d\n", v32)
	v32 = mem.G(5, 3)
	fmt.Printf("-- Asserting return... mem.G(5, 3) --> %d\n", v32)
	mem.PtrConvert()
	fmt.Printf("-- Invoking... mem.PtrConvert()\n")
	v32 = gc.Alloc(128, 64)
	fmt.Printf("-- Asserting return... gc.Alloc(128, 64) --> %d\n", v32)
	v32 = gc.Alloc(64, 32)
	fmt.Printf("-- Asserting return... gc.Alloc(64, 32) --> %d\n", v32)
	newstuff.PtrConvert()
	fmt.Printf("-- Invoking... newstuff.PtrConvert()\n")
	fmt.Printf("Tests complete\n")
}
