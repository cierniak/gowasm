package main

import "gowasm/rt/v8"

func main() {
	a := [...]byte{'h', 'e', 'l', 'l', 'o', 0}
	p := &a[0]
	v8.Puts(p)
}
