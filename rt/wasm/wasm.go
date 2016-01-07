package wasm

import (
	"fmt"
)

func Print_int32(n int32) {
	fmt.Printf("%d : i32\n", n)
}

func Print_int64(n int64) {
	fmt.Printf("%d : i64\n", n)
}

func Puts(p *byte) int {
	fmt.Printf("%v : pointer (puts)\n", *p)
	return 0
}
